# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-01-01

### Added

- User registration and authentication
- Domain management (add, edit, delete)
- Automatic WHOIS lookup and expiry detection
- Dashboard with color-coded progress bars
- Email notifications with verification flow
- Telegram notification support
- Customizable notification thresholds
- Background job for automatic domain checking
- Docker support for deployment
- Comprehensive test suite

### Features

- Visual countdown bars showing days remaining
- Color-coded warnings: green (>60), yellow (30-60), orange (14-30), red (<14), pulsing (<7)
- Email verification to prevent spam
- Multi-channel notifications (email, Telegram)
- Modular notification system for easy extension
- Session-based authentication
- Responsive web UI with HTMX

### Tech Stack

- Go 1.21+ with Echo v4
- SQLite3 for storage
- HTMX for dynamic updates
- Standard library for WHOIS
- Docker for deployment

## [0.1.0] - 2024-01-01

### Added

- Initial project structure
- Basic Echo server setup
- Database schema

[1.0.0]: https://github.com/CaffeinatedTech/domain-minder/compare/v0.1.0...v1.0.0
[0.1.0]: https://github.com/CaffeinatedTech/domain-minder/releases/tag/v0.1.0
