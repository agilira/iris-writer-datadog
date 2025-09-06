# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-09-06

### Added
- Initial release of iris-writer-datadog
- Datadog Logs API writer implementation for iris logging library
- Integration with public iris v1.0.0 from github.com/agilira/iris
- Complete test suite with unit tests
- Comprehensive documentation and README
- Support for structured logging to Datadog Logs API
- Configurable batch size and timeout settings
- Error handling and retry mechanisms
- Thread-safe implementation
- Native Datadog integration with:
  - Service, environment, version tagging
  - Custom tags support
  - Hostname identification
  - Source tagging
  - Proper log level mapping to Datadog severity levels

### Security
- Initial security audit completed
- No known vulnerabilities
- Secure API key handling

### Documentation
- Complete API documentation
- Usage examples and integration guide
- Contributing guidelines and code of conduct
