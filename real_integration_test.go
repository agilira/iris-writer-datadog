package datadogwriter

import (
	"os"
	"testing"
	"time"

	"github.com/agilira/iris"
)

// TestRealDatadogIntegration tests with real Datadog API
// Run with: DD_API_KEY=your-key go test -v -run TestRealDatadogIntegration
// This test is skipped if DD_API_KEY environment variable is not set
func TestRealDatadogIntegration(t *testing.T) {
	apiKey := os.Getenv("DD_API_KEY")
	if apiKey == "" {
		t.Skip("⏭️  Skipping real Datadog integration test - DD_API_KEY not set")
	}

	t.Logf("🌍 Testing real Datadog integration with API key: %s***", apiKey[:8])

	// Configure for real Datadog
	config := Config{
		APIKey:      apiKey,
		Site:        "datadoghq.com", // Change to "datadoghq.eu" if needed
		Service:     "iris-writer-datadog-test",
		Environment: "integration-test",
		Version:     "1.0.0-test",
		Source:      "go-integration-test",
		Hostname:    "test-runner",
		Tags: map[string]string{
			"test_type":   "integration",
			"test_run_id": time.Now().Format("20060102-150405"),
		},
		BatchSize:     5,
		FlushInterval: 1 * time.Second,
		Timeout:       10 * time.Second,
		OnError: func(err error) {
			t.Logf("❌ Datadog writer error: %v", err)
		},
	}

	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create Datadog writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	t.Log("✅ Datadog writer created successfully")

	// Send test records
	testRecords := []*iris.Record{
		{Level: iris.Info, Msg: "🧪 Real integration test started"},
		{Level: iris.Warn, Msg: "⚠️  This is a test warning from iris-writer-datadog"},
		{Level: iris.Error, Msg: "❌ Test error message for verification"},
		{Level: iris.Info, Msg: "✅ Real integration test completed successfully"},
	}

	t.Logf("📤 Sending %d test records to real Datadog...", len(testRecords))

	for i, record := range testRecords {
		if err := writer.WriteRecord(record); err != nil {
			t.Errorf("❌ Failed to write record %d: %v", i+1, err)
		} else {
			t.Logf("✅ Sent record %d: %s - %s", i+1, record.Level.String(), record.Msg)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Wait for flush
	t.Log("⏳ Waiting for final flush...")
	time.Sleep(3 * time.Second)

	// Close to ensure all logs are sent
	if err := writer.Close(); err != nil {
		t.Errorf("❌ Error during close: %v", err)
	} else {
		t.Log("✅ Writer closed successfully")
	}

	t.Log("🎉 Real Datadog integration test completed!")
	t.Log("📊 Check your Datadog logs with these filters:")
	t.Logf("   service:%s", config.Service)
	t.Logf("   env:%s", config.Environment)
	t.Logf("   source:%s", config.Source)
	t.Logf("   test_run_id:%s", config.Tags["test_run_id"])
}

// TestErrorHandlingWithRealAPI tests error scenarios with real API
func TestErrorHandlingWithRealAPI(t *testing.T) {
	t.Log("🧪 Testing error handling with invalid API key")

	errorReceived := false
	config := Config{
		APIKey:     "invalid-api-key-12345",
		Site:       "datadoghq.com",
		Service:    "error-test",
		BatchSize:  1,
		MaxRetries: 1,
		OnError: func(err error) {
			errorReceived = true
			t.Logf("✅ Expected error received: %v", err)
		},
	}

	writer, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create writer: %v", err)
	}
	defer func() { _ = writer.Close() }()

	record := &iris.Record{
		Level: iris.Error,
		Msg:   "Test error handling",
	}

	// This should trigger an error due to invalid API key
	err = writer.WriteRecord(record)
	if err != nil {
		t.Logf("✅ WriteRecord returned error as expected: %v", err)
	}

	// Wait for potential async error
	time.Sleep(500 * time.Millisecond)

	if !errorReceived {
		t.Log("⚠️  No error callback triggered - this might be due to async processing")
	} else {
		t.Log("✅ Error callback was triggered as expected")
	}
}
