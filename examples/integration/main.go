package main

import (
	"log"
	"os"
	"time"

	"github.com/agilira/iris"
	datadogwriter "github.com/agilira/iris-writer-datadog"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("DD_API_KEY")
	if apiKey == "" {
		log.Fatal("DD_API_KEY environment variable is required")
	}

	// Configure Datadog writer
	config := datadogwriter.Config{
		APIKey:      apiKey,
		Site:        "datadoghq.com", // Use "datadoghq.eu" for EU
		Service:     "iris-datadog-example",
		Environment: "development",
		Version:     "1.0.0",
		Source:      "go",
		Hostname:    "example-host",
		Tags: map[string]string{
			"team":      "backend",
			"component": "example",
		},
		BatchSize:         100, // Smaller batch for demo
		FlushInterval:     2 * time.Second,
		Timeout:           10 * time.Second,
		EnableCompression: true, // Enable gzip compression to reduce bandwidth
		OnError: func(err error) {
			log.Printf("Datadog writer error: %v", err)
		},
	}

	writer, err := datadogwriter.New(config)
	if err != nil {
		log.Fatalf("Failed to create Datadog writer: %v", err)
	}
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("Error closing writer: %v", err)
		}
	}()

	// Send some example logs directly using the writer
	log.Println("Sending logs to Datadog...")

	// Create test records manually
	records := []*iris.Record{
		{
			Level: iris.Debug,
			Msg:   "This is a debug message",
		},
		{
			Level: iris.Info,
			Msg:   "Application started successfully",
		},
		{
			Level: iris.Warn,
			Msg:   "This is a warning message",
		},
		{
			Level: iris.Error,
			Msg:   "This is an error message",
		},
		{
			Level: iris.Info,
			Msg:   "User action performed with user_id=12345",
		},
		{
			Level: iris.Info,
			Msg:   "Request processed: POST /api/users - 201",
		},
		{
			Level: iris.Error,
			Msg:   "Database connection failed: postgresql://db.example.com:5432",
		},
	}

	// Send records to the writer
	for i, record := range records {
		if err := writer.WriteRecord(record); err != nil {
			log.Printf("❌ Failed to write record %d: %v", i+1, err)
		} else {
			log.Printf("✅ Sent record %d: %s - %s",
				i+1, record.Level.String(), record.Msg)
		}

		// Small delay between records
		time.Sleep(200 * time.Millisecond)
	}

	// Wait a moment for final flush
	time.Sleep(3 * time.Second)

	log.Println("Example completed. Check your Datadog logs!")
}
