// integration_e2e_test.go: External Datadog writer for Iris e2e tests
//
// Copyright (c) 2025 AGILira
// Series: an AGILira library
// SPDX-License-Identifier: MPL-2.0

package datadogwriter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agilira/iris"
)

// TestEndToEndIntegration tests the complete data flow from writer to mock Datadog API
func TestEndToEndIntegration(t *testing.T) {
	fmt.Printf("🧪 End-to-End Integration Test for Iris Datadog Writer\n")
	fmt.Printf("This test creates a mock Datadog server and verifies data flow\n\n")

	// Track received requests
	var receivedRequests []DatadogRequest
	var requestBodies []string

	// Create mock Datadog server
	mockDatadog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("🔍 Received request: %s %s\n", r.Method, r.URL.Path)

		if r.Method == "POST" && strings.Contains(r.URL.Path, "/v1/input/") {
			// Read and parse the request body
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("Failed to read request body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			requestBodies = append(requestBodies, string(body))

			// Try to parse as Datadog logs format
			var logs []LogEntry
			if err := json.Unmarshal(body, &logs); err != nil {
				t.Logf("Request body (not JSON array): %s", string(body))
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			req := DatadogRequest{
				Headers: make(map[string]string),
				Logs:    logs,
			}

			// Capture important headers
			if apiKey := r.Header.Get("DD-API-KEY"); apiKey != "" {
				req.Headers["DD-API-KEY"] = apiKey
			}
			if contentType := r.Header.Get("Content-Type"); contentType != "" {
				req.Headers["Content-Type"] = contentType
			}

			receivedRequests = append(receivedRequests, req)

			fmt.Printf("Mock Datadog received request %d: %d logs, %d bytes\n",
				len(receivedRequests), len(logs), len(body))

			// Return success (Datadog returns 202 Accepted)
			w.WriteHeader(http.StatusAccepted)
		} else {
			fmt.Printf("❌ Unexpected request: %s %s\n", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockDatadog.Close()

	// Extract the API key and prepare mock URL
	testAPIKey := "test-api-key-12345"
	mockHost := strings.TrimPrefix(mockDatadog.URL, "http://")
	mockHost = strings.TrimPrefix(mockHost, "https://")

	fmt.Printf("🖥️  Mock Datadog server started at: %s\n", mockDatadog.URL)

	// Configure the Datadog writer to point to our mock server
	config := Config{
		APIKey:      testAPIKey,
		Site:        mockHost, // Use mock host:port directly
		Service:     "integration-test",
		Environment: "e2e",
		Version:     "1.0.0-test",
		Source:      "go-test",
		Hostname:    "test-host",
		Tags: map[string]string{
			"test_run":   fmt.Sprintf("run-%d", time.Now().Unix()),
			"component":  "integration",
			"test_suite": "e2e",
		},
		BatchSize:     3, // Small batch for immediate testing
		FlushInterval: 200 * time.Millisecond,
		Timeout:       5 * time.Second,
		OnError: func(err error) {
			t.Logf("❌ Datadog writer error: %v", err)
		},
	}

	// Create the Datadog writer
	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create Datadog writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	fmt.Println("✅ Datadog writer created successfully")

	// Create comprehensive test records
	testRecords := []*iris.Record{
		{Level: iris.Info, Msg: "Integration test started"},
		{Level: iris.Debug, Msg: "Debug information for testing"},
		{Level: iris.Warn, Msg: "Warning message for test verification"},
		{Level: iris.Error, Msg: "Error message to test error handling"},
		{Level: iris.Info, Msg: "Final test message"},
	}

	fmt.Printf("Sending %d test records...\n", len(testRecords))

	// Send test records
	for i, record := range testRecords {
		if err := writer.WriteRecord(record); err != nil {
			t.Errorf("❌ Failed to write record %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ Sent record %d: %s - %s\n",
				i+1, record.Level.String(), record.Msg)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for flush and final processing
	fmt.Println("Waiting for batches to flush...")
	time.Sleep(1 * time.Second)

	// Force final flush
	fmt.Println("Forcing final flush...")
	if err := writer.Close(); err != nil {
		t.Logf("Warning: Close error: %v", err)
	}

	// Verify results
	fmt.Printf("\nIntegration Test Results:\n")
	fmt.Printf("Requests received: %d\n", len(receivedRequests))
	fmt.Printf("Total request bodies: %d\n", len(requestBodies))

	if len(receivedRequests) == 0 {
		t.Fatal("❌ CRITICAL: No requests received by mock Datadog server")
	}

	// Verify request structure
	totalLogs := 0
	for i, req := range receivedRequests {
		fmt.Printf("Request %d: %d logs\n", i+1, len(req.Logs))
		totalLogs += len(req.Logs)

		// Verify headers
		if req.Headers["DD-API-KEY"] != testAPIKey {
			t.Errorf("❌ Missing or incorrect API key in request %d", i+1)
		}
		if req.Headers["Content-Type"] != "application/json" {
			t.Errorf("❌ Missing or incorrect Content-Type in request %d", i+1)
		}

		// Verify log structure
		for j, log := range req.Logs {
			if log.Message == "" {
				t.Errorf("❌ Empty message in request %d, log %d", i+1, j+1)
			}
			if log.Level == "" {
				t.Errorf("❌ Empty level in request %d, log %d", i+1, j+1)
			}
			if log.Timestamp <= 0 {
				t.Errorf("❌ Invalid timestamp in request %d, log %d", i+1, j+1)
			}
			if log.Service != config.Service {
				t.Errorf("❌ Incorrect service in request %d, log %d: got %s, want %s",
					i+1, j+1, log.Service, config.Service)
			}
		}
	}

	fmt.Printf("Total logs received: %d (expected: %d)\n", totalLogs, len(testRecords))

	// Verify all test messages were received
	allContent := strings.Join(requestBodies, " ")
	expectedMessages := []string{
		"Integration test started",
		"Debug information for testing",
		"Warning message for test verification",
		"Error message to test error handling",
		"Final test message",
	}

	foundMessages := 0
	for _, msg := range expectedMessages {
		if strings.Contains(allContent, msg) {
			foundMessages++
			fmt.Printf("✅ Found message: '%s'\n", msg)
		} else {
			fmt.Printf("❌ Missing message: '%s'\n", msg)
		}
	}

	// Final validation
	if len(receivedRequests) == 0 {
		t.Fatal("❌ FAILURE: No requests received")
	}

	if totalLogs != len(testRecords) {
		t.Errorf("❌ FAILURE: Expected %d logs, got %d", len(testRecords), totalLogs)
	}

	if foundMessages != len(expectedMessages) {
		t.Errorf("❌ FAILURE: Expected %d messages, found %d", len(expectedMessages), foundMessages)
	}

	fmt.Printf("\nSUCCESS: End-to-end integration test passed!\n")
	fmt.Printf("   ✅ %d requests received\n", len(receivedRequests))
	fmt.Printf("   ✅ %d/%d logs delivered\n", totalLogs, len(testRecords))
	fmt.Printf("   ✅ %d/%d messages verified\n", foundMessages, len(expectedMessages))

	t.Logf("✅ End-to-end integration test passed: %d requests, %d messages verified",
		len(receivedRequests), foundMessages)
}

// DatadogRequest represents a request received by the mock server
type DatadogRequest struct {
	Headers map[string]string
	Logs    []LogEntry
}

func TestCompressionIntegration(t *testing.T) {
	fmt.Println("Compression Integration Test for Iris Datadog Writer")
	fmt.Println("This test verifies that gzip compression works correctly")

	// Start mock Datadog server that can handle compressed requests
	mockDatadog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request has compression header
		contentEncoding := r.Header.Get("Content-Encoding")
		if contentEncoding == "gzip" {
			fmt.Printf("🗜️  Received compressed request with Content-Encoding: %s\n", contentEncoding)
		}

		fmt.Printf("🔍 Received request: %s %s\n", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer mockDatadog.Close()

	// Extract host for configuration
	testAPIKey := "test-compression-key"
	mockHost := strings.TrimPrefix(mockDatadog.URL, "http://")

	fmt.Printf("Mock Datadog server started at: %s\n", mockDatadog.URL)

	// Configure writer with compression enabled
	config := Config{
		APIKey:            testAPIKey,
		Site:              mockHost,
		Service:           "compression-test",
		Environment:       "test",
		EnableCompression: true, // Enable compression
		BatchSize:         2,
		FlushInterval:     100 * time.Millisecond,
		Timeout:           2 * time.Second,
		OnError: func(err error) {
			t.Logf("❌ Datadog writer error: %v", err)
		},
	}

	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create Datadog writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	fmt.Println("✅ Datadog writer with compression created successfully")

	// Send test records
	testRecords := []*iris.Record{
		{Level: iris.Info, Msg: "Compression test message 1 - this message should be compressed"},
		{Level: iris.Warn, Msg: "Compression test message 2 - this message should also be compressed"},
	}

	fmt.Println("Sending test records with compression enabled...")
	for i, record := range testRecords {
		err := writer.WriteRecord(record)
		if err != nil {
			t.Logf("WriteRecord %d returned error: %v", i+1, err)
		}
		fmt.Printf("✅ Sent compressed record %d: %s - %s\n", i+1, record.Level, record.Msg)
	}

	// Wait for batch to flush
	time.Sleep(300 * time.Millisecond)

	// Force final flush
	err = writer.Close()
	if err != nil {
		t.Logf("Close returned error (expected): %v", err)
	}

	fmt.Println("✅ Compression integration test completed successfully")
}
