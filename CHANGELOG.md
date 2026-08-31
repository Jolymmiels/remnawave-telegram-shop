# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Migrated Remnawave API client to v3.4.1:
  - User lookup by Telegram ID now uses the `telegramId` filter of `GET /api/users/stream` (the legacy `/api/users/by-telegram-id` endpoint was removed)
  - User updates are sent by numeric `id` instead of `uuid` (`PATCH /api/users` requires exactly one of `id`/`username` since v3)
  - `telegramId` in create/update bodies and `description` in the update body are arrays per the v3.4.1 schema

### Added
- New `customer.remnawave_id` column (migration 000006): user sync now stores the panel numeric user ID for every synced customer

## [3.4.1] - 2025-11-08

### Added
- `TRIAL_REMNAWAVE_TAG` environment variable to assign different tags to trial users in Remnawave
- Trial users can now be tagged separately from paying customers for better tracking and resource management

### Changed
- User tag assignment now always applies trial-specific or regular tags based on user type
- Trial users get assigned `TRIAL_REMNAWAVE_TAG` (or fallback to `REMNAWAVE_TAG`) when created or updated
- Regular paying users continue to receive `REMNAWAVE_TAG` on all operations

### Documentation
- Added `TRIAL_REMNAWAVE_TAG` to environment variables table in README
- Updated `.env.sample` with trial tag configuration example
## [3.4.0] - 2025-11-08

### Added
- `TRIAL_INTERNAL_SQUADS` environment variable for configuring separate internal squads for trial users
- `TRIAL_EXTERNAL_SQUAD_UUID` environment variable for configuring separate external squad for trial users
- Trial user squad configuration with automatic fallback to regular squad settings when trial-specific settings are not provided
- Support for isolated squad assignment of trial users from regular paid users

### Changed
- Trial user creation now supports isolated squad assignment from regular paid users
- `CreateOrUpdateUser()`, `createUser()`, and `updateUser()` methods now accept `isTrialUser` parameter to determine squad selection
- Payment service now passes trial user flag when activating trial subscriptions

### Documentation
- Added comprehensive Trial Squad Configuration section in README explaining use cases and behavior
- Updated environment variables table with new trial squad configuration parameters
- Added examples for trial squad UUID configuration

## [3.3.3] - 2025-11-07

### Added
- `REQUIRE_PAID_PURCHASE_FOR_STARS` environment variable to gate-keep Telegram Stars payment method
- Telegram Stars payment option now requires at least one successful cryptocurrency or card payment to be available
- New database repository method `FindSuccessfulPaidPurchaseByCustomer()` to check user payment history

### Changed
- Telegram Stars button is now conditionally displayed in payment method selection based on user's payment history
- Users without prior crypto or card payments will only see available payment methods

### Security
- Enhanced payment flow to prevent Telegram Stars abuse by new unverified users

## [3.3.2] - 2025-11-05

### Added
- `WHITELISTED_TELEGRAM_IDS` environment variable to whitelist users by Telegram ID (comma-separated list)
- Whitelisted users bypass all suspicious user checks

### Changed
- Improved suspicious user detection: now checks for dangerous keyword combinations instead of individual keywords
  - Allows legitimate project accounts like @CompanySupportAdmin to pass validation
  - Maintains detection of actual phishing accounts (e.g., @TelegramSupport, @ServiceSupport)
  - Detects combinations: telegram+support, telegram+admin, service+support, system+admin, security+admin

### Fixed
- False positives in suspicious user detection for project accounts with service-related names

## [3.3.1] - 2025-11-05

### Added
- `BLOCKED_TELEGRAM_IDS` environment variable for blocking users by Telegram ID (comma-separated list)
- User blacklist functionality in suspicious user filter middleware

### Changed
- SuspiciousUserFilterMiddleware now checks blacklist before suspicious name pattern validation for better security control

### Documentation
- Added `BLOCKED_TELEGRAM_IDS` to environment variables table in README
- Updated `.env.sample` with blocked telegram IDs configuration example

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
