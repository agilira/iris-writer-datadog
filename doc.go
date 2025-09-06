// Package datadogwriter provides a Datadog Logs API writer for the Iris logging library.
//
// This package implements the iris.SyncWriter interface to enable high-performance
// log shipping to Datadog. It supports batching, retry logic, and comprehensive
// configuration options for production use.
//
// # Basic Usage
//
//	config := datadogwriter.Config{
//		APIKey:      "your-datadog-api-key",
//		Site:        "datadoghq.com", // or "datadoghq.eu"
//		Service:     "my-service",
//		Environment: "production",
//		Version:     "1.0.0",
//	}
//
//	writer, err := datadogwriter.New(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer writer.Close()
//
//	logger := iris.New(iris.WithSyncWriter(writer))
//	logger.Info("Hello from Iris to Datadog!")
//
// # Configuration
//
// The Config struct provides extensive customization options:
//
//   - APIKey: Required Datadog API key for authentication
//   - Site: Datadog site (datadoghq.com, datadoghq.eu, etc.)
//   - Service, Environment, Version: Standard Datadog tags
//   - BatchSize: Number of logs to batch before sending (default: 1000)
//   - FlushInterval: Maximum time before flushing incomplete batches (default: 1s)
//   - Timeout: HTTP request timeout (default: 10s)
//   - OnError: Optional error callback function
//   - MaxRetries: Number of retry attempts (default: 3)
//   - RetryDelay: Delay between retries (default: 100ms)
//
// # Performance
//
// This writer is optimized for high-throughput logging:
//
//   - Batches multiple log entries in single HTTP requests
//   - Uses time-based flushing to ensure timely delivery
//   - Employs efficient JSON marshaling for Datadog's format
//   - Implements retry logic with exponential backoff
//   - Thread-safe for concurrent logging operations
//
// # Error Handling
//
// The writer includes comprehensive error handling:
//
//   - Configurable retry logic for transient failures
//   - Optional error callback for monitoring integration
//   - Graceful degradation on persistent failures
//   - Proper resource cleanup on shutdown
//
// # Integration
//
// This package integrates seamlessly with the Iris ecosystem:
//
//	iris (core) → SyncWriter interface → iris-writer-datadog → Datadog Logs API
//
// The external architecture ensures zero dependencies in core Iris while
// providing powerful log aggregation capabilities for Datadog users.
package datadogwriter
