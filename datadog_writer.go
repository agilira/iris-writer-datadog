package datadogwriter

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/agilira/go-timecache"
	"github.com/agilira/iris"
)

// Writer implements iris.SyncWriter for Datadog Logs API
type Writer struct {
	config     Config
	client     *http.Client
	buffer     []LogEntry
	mutex      sync.Mutex
	timer      *time.Timer
	timerMutex sync.Mutex // Protects timer access
	closed     bool       // Tracks if writer is closed
}

// Config holds the configuration for the Datadog writer
type Config struct {
	// APIKey is the Datadog API key for authentication
	APIKey string

	// Site is the Datadog site (e.g., "datadoghq.com", "datadoghq.eu")
	Site string

	// Service name to tag logs with
	Service string

	// Environment to tag logs with
	Environment string

	// Version to tag logs with
	Version string

	// Source to tag logs with (e.g., "go", "application")
	Source string

	// Hostname to tag logs with
	Hostname string

	// Additional tags to attach to all logs
	Tags map[string]string

	// BatchSize is the maximum number of log entries to batch before sending
	BatchSize int

	// FlushInterval is the maximum time to wait before flushing incomplete batches
	FlushInterval time.Duration

	// Timeout for HTTP requests to Datadog
	Timeout time.Duration

	// OnError is an optional callback for handling errors
	OnError func(error)

	// MaxRetries is the number of retry attempts for failed requests
	MaxRetries int

	// RetryDelay is the delay between retry attempts
	RetryDelay time.Duration

	// EnableCompression enables gzip compression for HTTP requests to reduce bandwidth
	EnableCompression bool
}

// LogEntry represents a single log entry for Datadog
type LogEntry struct {
	Timestamp int64          `json:"timestamp"`
	Level     string         `json:"status"`
	Message   string         `json:"message"`
	Service   string         `json:"service,omitempty"`
	Source    string         `json:"ddsource,omitempty"`
	Tags      string         `json:"ddtags,omitempty"`
	Hostname  string         `json:"hostname,omitempty"`
	Env       string         `json:"env,omitempty"`
	Version   string         `json:"version,omitempty"`
	Fields    map[string]any `json:",inline"`
}

// New creates a new Datadog writer with the given configuration
func New(config Config) (*Writer, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Set defaults
	if config.Site == "" {
		config.Site = "datadoghq.com"
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 1000
	}
	if config.FlushInterval <= 0 {
		config.FlushInterval = time.Second
	}
	if config.Timeout <= 0 {
		config.Timeout = 10 * time.Second
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 100 * time.Millisecond
	}
	if config.Source == "" {
		config.Source = "go"
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	writer := &Writer{
		config: config,
		client: client,
		buffer: make([]LogEntry, 0, config.BatchSize),
	}

	writer.startFlushTimer()
	return writer, nil
}

// WriteRecord implements iris.SyncWriter
func (w *Writer) WriteRecord(record *iris.Record) error {
	entry := w.buildLogEntry(record)

	w.mutex.Lock()
	w.buffer = append(w.buffer, entry)
	shouldFlush := len(w.buffer) >= w.config.BatchSize
	w.mutex.Unlock()

	if shouldFlush {
		return w.flush()
	}
	return nil
}

// Close flushes remaining logs and shuts down the writer
func (w *Writer) Close() error {
	w.timerMutex.Lock()
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.closed = true
	w.timerMutex.Unlock()

	return w.flush()
}

func (w *Writer) buildLogEntry(record *iris.Record) LogEntry {
	entry := LogEntry{
		Timestamp: timecache.CachedTimeNano() / 1000000, // Convert to milliseconds
		Level:     mapLevel(record.Level),
		Message:   record.Msg,
		Service:   w.config.Service,
		Source:    w.config.Source,
		Hostname:  w.config.Hostname,
		Env:       w.config.Environment,
		Version:   w.config.Version,
		Fields:    make(map[string]any),
	}

	// Build tags string
	if len(w.config.Tags) > 0 {
		entry.Tags = w.buildTagsString()
	}

	return entry
}

func (w *Writer) buildTagsString() string {
	if len(w.config.Tags) == 0 {
		return ""
	}

	var tags []string
	for key, value := range w.config.Tags {
		tags = append(tags, fmt.Sprintf("%s:%s", key, value))
	}

	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += tag
	}
	return result
}

func (w *Writer) flush() error {
	w.mutex.Lock()
	if len(w.buffer) == 0 {
		w.mutex.Unlock()
		return nil
	}

	entries := make([]LogEntry, len(w.buffer))
	copy(entries, w.buffer)
	w.buffer = w.buffer[:0]
	w.mutex.Unlock()

	return w.sendToDatadog(entries)
}

func (w *Writer) sendToDatadog(entries []LogEntry) error {
	payload, err := json.Marshal(entries)
	if err != nil {
		w.handleError(fmt.Errorf("failed to marshal log entries: %w", err))
		return err
	}

	// Apply compression if enabled
	var body []byte
	var contentEncoding string
	if w.config.EnableCompression {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(payload); err != nil {
			w.handleError(fmt.Errorf("failed to compress payload: %w", err))
			return err
		}
		if err := gz.Close(); err != nil {
			w.handleError(fmt.Errorf("failed to close gzip writer: %w", err))
			return err
		}
		body = buf.Bytes()
		contentEncoding = "gzip"
	} else {
		body = payload
	}

	// Build the Datadog intake URL
	var url string
	if strings.Contains(w.config.Site, "127.0.0.1") || strings.Contains(w.config.Site, "localhost") {
		// For local testing/development
		url = fmt.Sprintf("http://%s/v1/input/%s", w.config.Site, w.config.APIKey)
	} else {
		// Standard Datadog endpoint
		url = fmt.Sprintf("https://http-intake.logs.%s/v1/input/%s", w.config.Site, w.config.APIKey)
	}

	var lastErr error
	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(w.config.RetryDelay * time.Duration(attempt))
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("DD-API-KEY", w.config.APIKey)
		if contentEncoding != "" {
			req.Header.Set("Content-Encoding", contentEncoding)
		}

		resp, err := w.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			continue
		}

		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("datadog API error: status %d", resp.StatusCode)

		// Don't retry on client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			break
		}
	}

	w.handleError(lastErr)
	return lastErr
}

func (w *Writer) startFlushTimer() {
	w.timerMutex.Lock()
	defer w.timerMutex.Unlock()

	if w.closed {
		return
	}

	w.timer = time.AfterFunc(w.config.FlushInterval, func() {
		_ = w.flush()
		w.startFlushTimer()
	})
}

func (w *Writer) handleError(err error) {
	if w.config.OnError != nil && err != nil {
		w.config.OnError(err)
	}
}

func mapLevel(level iris.Level) string {
	switch level {
	case iris.Debug:
		return "debug"
	case iris.Info:
		return "info"
	case iris.Warn:
		return "warn"
	case iris.Error:
		return "error"
	case iris.DPanic:
		return "critical"
	case iris.Panic:
		return "emergency"
	case iris.Fatal:
		return "critical"
	default:
		return "info"
	}
}
