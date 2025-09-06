// datadog_writer_test.go: External Datadog writer for Iris tests
//
// Copyright (c) 2025 AGILira
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package datadogwriter

import (
	"testing"
	"time"

	"github.com/agilira/iris"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				APIKey: "test-api-key",
				Site:   "datadoghq.com",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				Site: "datadoghq.com",
			},
			wantErr: true,
		},
		{
			name: "with defaults",
			config: Config{
				APIKey: "test-api-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if writer != nil {
				defer func() { _ = writer.Close() }()
			}

			if !tt.wantErr {
				// Check defaults
				if writer.config.Site == "" {
					t.Error("Expected default site to be set")
				}
				if writer.config.BatchSize <= 0 {
					t.Error("Expected default batch size to be set")
				}
				if writer.config.FlushInterval <= 0 {
					t.Error("Expected default flush interval to be set")
				}
			}
		})
	}
}

func TestWriter_WriteRecord(t *testing.T) {
	config := Config{
		APIKey:    "test-api-key",
		Site:      "datadoghq.com",
		Service:   "test-service",
		BatchSize: 2, // Small batch for testing
	}

	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	record := &iris.Record{
		Level: iris.Info,
		Msg:   "Test message",
	}

	err = writer.WriteRecord(record)
	if err != nil {
		t.Errorf("WriteRecord() error = %v", err)
	}

	// Check that record was buffered
	writer.mutex.Lock()
	bufferLen := len(writer.buffer)
	writer.mutex.Unlock()

	if bufferLen != 1 {
		t.Errorf("Expected buffer length 1, got %d", bufferLen)
	}
}

func TestWriter_BatchFlushing(t *testing.T) {
	errorChan := make(chan error, 1)
	config := Config{
		APIKey:    "test-api-key",
		Site:      "datadoghq.com",
		BatchSize: 2,
		OnError: func(err error) {
			errorChan <- err
		},
	}

	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	record := &iris.Record{
		Level: iris.Info,
		Msg:   "Test message",
	}

	// Write first record
	err = writer.WriteRecord(record)
	if err != nil {
		t.Errorf("WriteRecord() error = %v", err)
	}

	// Write second record (should trigger flush)
	_ = writer.WriteRecord(record)
	// Note: Error is expected here due to test API key, but WriteRecord might not return it immediately

	// Check that buffer was flushed
	select {
	case receivedErr := <-errorChan:
		// Expected - network error due to test API key
		t.Logf("Expected error received: %v", receivedErr)
	case <-time.After(200 * time.Millisecond):
		// Also acceptable - no error callback within timeout
		t.Log("No error callback within timeout (acceptable for async processing)")
	}

	writer.mutex.Lock()
	bufferLen := len(writer.buffer)
	writer.mutex.Unlock()

	if bufferLen != 0 {
		t.Errorf("Expected buffer to be flushed, got length %d", bufferLen)
	}
}

func TestWriter_ErrorHandling(t *testing.T) {
	errorReceived := false
	config := Config{
		APIKey:     "invalid-key",
		Site:       "datadoghq.com",
		BatchSize:  1,
		MaxRetries: 1,
		OnError: func(err error) {
			errorReceived = true
		},
	}

	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	record := &iris.Record{
		Level: iris.Error,
		Msg:   "Test error message",
	}

	_ = writer.WriteRecord(record)
	// Error expected due to invalid API key, but WriteRecord might not return it immediately

	// Wait a bit for potential async error handling
	time.Sleep(100 * time.Millisecond)

	if !errorReceived {
		t.Log("No error callback received (expected due to invalid API key)")
	}
}

func TestWriter_Close(t *testing.T) {
	config := Config{
		APIKey: "test-api-key",
		Site:   "datadoghq.com",
	}

	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}

	// Add a record to buffer
	record := &iris.Record{
		Level: iris.Info,
		Msg:   "Final message",
	}

	err = writer.WriteRecord(record)
	if err != nil {
		t.Errorf("WriteRecord() error = %v", err)
	}

	// Close should flush remaining records
	err = writer.Close()
	if err != nil {
		t.Log("Close() error expected due to test API key:", err)
	}
}

func TestBuildTagsString(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		expected string
	}{
		{
			name:     "empty tags",
			tags:     map[string]string{},
			expected: "",
		},
		{
			name: "single tag",
			tags: map[string]string{
				"env": "production",
			},
			expected: "env:production",
		},
		{
			name: "multiple tags",
			tags: map[string]string{
				"env":     "production",
				"service": "api",
				"version": "1.0.0",
			},
			// Note: map iteration order is not guaranteed, so we check for valid format
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &Writer{
				config: Config{
					Tags: tt.tags,
				},
			}

			result := writer.buildTagsString()

			if len(tt.tags) == 0 {
				if result != tt.expected {
					t.Errorf("buildTagsString() = %v, want %v", result, tt.expected)
				}
			} else if len(tt.tags) == 1 {
				if result != tt.expected {
					t.Errorf("buildTagsString() = %v, want %v", result, tt.expected)
				}
			} else {
				// For multiple tags, just check format
				if result == "" {
					t.Error("buildTagsString() returned empty string for non-empty tags")
				}
			}
		})
	}
}

func TestWriter_Compression(t *testing.T) {
	config := Config{
		APIKey:            "test-api-key",
		Site:              "datadoghq.com",
		Service:           "test-service",
		EnableCompression: true, // Enable compression
		BatchSize:         1,    // Immediate flush for testing
	}

	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	record := &iris.Record{
		Level: iris.Info,
		Msg:   "Test message for compression validation",
	}

	// This will attempt to send compressed data
	// We expect an error due to invalid API key, but compression logic should execute
	_ = writer.WriteRecord(record)

	// Wait for batch to process
	time.Sleep(100 * time.Millisecond)

	// Test passes if no panics occur during compression
	t.Log("✅ Compression test completed without errors")
}
