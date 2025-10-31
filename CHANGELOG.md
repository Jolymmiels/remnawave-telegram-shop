# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [3.3.0] - 2025-10-31

### Added
- Application version management via ldflags (`Version`, `Commit`, `BuildDate` variables)
- Version information logging at application startup
- Version metadata in healthcheck endpoint response
- `EXTERNAL_SQUAD_UUID` configuration parameter for user creation and updates
- Development Docker build script (`build-dev.sh`) for easier local image creation
- Pagination helper support through remnawave-api-go v2.2.3

### Changed
- **Breaking:** Terminology refactored from "inbound" to "squad" throughout configuration and API integration
- Go version updated from 1.24 to 1.25.3
- Migrated to remnawave-api-go v2.2.3 with enhanced pagination support
- Build system improved with explicit git commit hash capture in Docker builds
- Environment variables `.env.sample` updated with new configuration options and documentation

### Fixed
- User language field preservation during sync patch update
- False positives in username filtering for better accuracy

### Documentation
- Added comprehensive documentation for `EXTERNAL_SQUAD_UUID` configuration parameter in README
- Updated README with new build scripts and version management information
- Added description of squad-based terminology changes

### Security
- Improved username validation filtering to reduce false positives while maintaining security

## [3.2.0] - 2025-01-08

### Added
- `DEFAULT_LANGUAGE` environment variable for configurable default bot language
- Support for setting default language to `en` (English) or `ru` (Russian)
- `build-release.sh` script for multi-platform Docker image building
- `purchase_test.go` test file for database purchase operations

### Fixed
- Dockerfile ARG duplication - replaced second `TARGETOS` with correct `TARGETARCH`
- Docker Compose restart policy improved from `always` to `unless-stopped`

### Changed
- Translation manager now accepts default language parameter during initialization
- Config initialization includes default language from environment variable

### Documentation
- Updated README.md with `DEFAULT_LANGUAGE` environment variable description
- Added usage examples for language configuration

## [3.1.4] - Previous Release

### Fixed
- Tribute payment processing issues

## [3.1.3] - Previous Release

### Fixed
- CryptoPay bot error in payment request handling

---

## Release Types

- **Added** for new features
- **Changed** for changes in existing functionality
- **Deprecated** for soon-to-be removed features
- **Removed** for now removed features
- **Fixed** for any bug fixes
- **Security** for vulnerability fixes

## Versioning

This project follows [Semantic Versioning](https://semver.org/):
- **MAJOR** version for incompatible API changes
- **MINOR** version for backwards-compatible functionality additions
- **PATCH** version for backwards-compatible bug fixes
