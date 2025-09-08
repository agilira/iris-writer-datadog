# Iris Writer for Datadog
### an AGILira library

[![CI](https://github.com/agilira/iris-writer-datadog/actions/workflows/ci.yml/badge.svg)](https://github.com/agilira/iris-writer-datadog/actions/workflows/ci.yml)
[![Security](https://img.shields.io/badge/security-gosec-brightgreen.svg)](https://github.com/agilira/iris-writer-datadog/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/agilira/iris-writer-datadog)](https://goreportcard.com/report/github.com/agilira/iris-writer-datadog)
[![Made For Iris](https://img.shields.io/badge/Made_for-Iris-pink)](https://github.com/agilira/iris)

External Datadog writer module for the Iris.

## Overview

This module implements the `iris.SyncWriter` interface to provide high-performance log shipping to Datadog Logs API. It's designed as an external module to keep the core Iris library dependency-free while enabling powerful log aggregation capabilities.

## Features

- **High Performance**: Uses timecache for optimized timestamp generation
- **Batching**: Configurable batch sizes and flush intervals
- **Resilience**: Built-in retry logic and error handling
- **Concurrent**: Safe for concurrent use with internal buffering
- **Datadog Integration**: Native support for Datadog tags, service, environment, and version

## Installation

```bash
go get github.com/agilira/iris-writer-datadog
```

## Usage

```go
package main

import (
    "log"
    "time"
    
    "github.com/agilira/iris"
    datadogwriter "github.com/agilira/iris-writer-datadog"
)

func main() {
    // Configure Datadog writer
    config := datadogwriter.Config{
        APIKey:      "your-datadog-api-key",
        Site:        "datadoghq.com", // or "datadoghq.eu"
        Service:     "my-service",
        Environment: "production",
        Version:     "1.0.0",
        Source:      "go",
        Hostname:    "web-server-01",
        Tags: map[string]string{
            "team": "backend",
            "component": "api",
        },
        BatchSize:     1000,
        FlushInterval: time.Second,
        Timeout:       10 * time.Second,
        OnError: func(err error) {
            log.Printf("Datadog writer error: %v", err)
        },
    }
    
    writer, err := datadogwriter.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer writer.Close()
    
    // Use with Iris logger
    logger := iris.New(iris.WithSyncWriter(writer))
    
    logger.Info("Hello from Iris to Datadog!")
}
```

## Configuration

- `APIKey`: Datadog API key for authentication (required)
- `Site`: Datadog site (default: "datadoghq.com", also supports "datadoghq.eu")
- `Service`: Service name to tag logs with
- `Environment`: Environment to tag logs with (e.g., "production", "staging")
- `Version`: Version to tag logs with
- `Source`: Source to tag logs with (default: "go")
- `Hostname`: Hostname to tag logs with
- `Tags`: Additional static tags to attach to all logs
- `BatchSize`: Number of records to batch before sending (default: 1000)
- `FlushInterval`: Maximum time to wait before flushing incomplete batches (default: 1s)
- `Timeout`: HTTP request timeout (default: 10s)
- `OnError`: Optional error callback function
- `MaxRetries`: Number of retry attempts (default: 3)
- `RetryDelay`: Delay between retries (default: 100ms)
- `EnableCompression`: Enable gzip compression for HTTP requests to reduce bandwidth (default: false)

## Datadog Integration

This writer sends logs directly to Datadog's Logs API with the following features:

- **Structured Logging**: JSON format optimized for Datadog
- **Automatic Tagging**: Service, environment, version, and custom tags
- **Level Mapping**: Iris log levels mapped to Datadog severity levels
- **Timestamp Precision**: Millisecond precision timestamps
- **Batch Optimization**: Efficient batching for high-throughput scenarios
- **Compression Support**: Optional gzip compression for reduced bandwidth usage

### High-Volume Logging

For applications with high log volume, enable compression to reduce network overhead:

```go
config := datadogwriter.Config{
    APIKey:            "your-datadog-api-key",
    Site:              "datadoghq.com",
    Service:           "my-service",
    EnableCompression: true, // Reduces bandwidth usage
    BatchSize:         1000, // Larger batches for efficiency
    FlushInterval:     5 * time.Second,
}
```

## Architecture

This module is part of the Iris modular ecosystem:

```
iris (core) → SyncWriter interface → iris-writer-datadog → Datadog Logs API
```

The external architecture ensures:
- Zero dependencies in core Iris
- Independent versioning and updates
- Modular functionality that can be added/removed as needed
- Performance optimizations specific to Datadog integration

iris-writer-datadog is licensed under the [Mozilla Public License 2.0](./LICENSE.md).

---

iris-writer-datadog • an AGILira library
