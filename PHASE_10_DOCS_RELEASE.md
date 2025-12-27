# Phase 10: Documentation & Release

## What Should Already Exist

- All Phase 1-9 deliverables complete
- All tests passing
- Server builds and runs successfully
- Project structure complete

## Context

This phase creates final documentation, Docker setup, deployment guide, and prepares the project for GitHub release. Update README with screenshots and deployment instructions. Create Docker Compose for development and Dockerfile for production.

## Agent Rules

1. Create files ONLY as specified below
2. Do NOT modify existing code - only add new files
3. Use MIT license as specified
4. Docker files must be production-ready
5. All documentation must be accurate and complete
6. Do NOT push to GitHub - only prepare files
7. Include screenshots description even if actual images not included
8. Provide clear deployment steps for production

## Tasks

### 10.1 Update README.md

Update `/home/adam/projects/domain-minder/README.md` with enhanced content:

```markdown
# Domain Minder

Never lose a domain to expiry again. Domain Minder monitors your domains and sends notifications when they're approaching expiration.

![Dashboard Preview](docs/dashboard.png)

## Features

- **Visual Dashboard**: Color-coded progress bars showing time remaining for each domain
- **Automatic Monitoring**: Background WHOIS checks every 6 hours
- **Multi-Channel Notifications**: Email and Telegram notifications
- **Email Verification**: Ensures notifications go to the right address
- **Customizable Thresholds**: Configure when you want to be notified
- **Self-Hosted**: Run on your own infrastructure

## Quick Start

### Prerequisites

- Go 1.21+
- SQLite3

### Installation

```bash
git clone https://github.com/CaffeinatedTech/domain-minder.git
cd domain-minder
go build -o domain-minder ./cmd/server/
./domain-minder
```

Access at http://localhost:9000

### Docker

```bash
docker compose up -d
```

## Configuration

Configure via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_PATH` | SQLite database file path | `data/domain_minder.db` |
| `PORT` | Server port | `9000` |
| `SESSION_SECRET` | Session encryption key | (required) |
| `CHECK_INTERVAL` | Domain check frequency | `6h` |
| `SMTP_HOST` | SMTP server hostname | (optional) |
| `SMTP_PORT` | SMTP port | `587` |
| `SMTP_USER` | SMTP username | (optional) |
| `SMTP_PASS` | SMTP password | (optional) |
| `SMTP_FROM` | From address for emails | `noreply@localhost` |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | (optional) |

### Example .env file

```bash
DB_PATH=data/domain_minder.db
PORT=9000
SESSION_SECRET=your-super-secret-key-change-me
CHECK_INTERVAL=6h

# Optional: Email notifications
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=your-email@example.com
SMTP_PASS=your-smtp-password
SMTP_FROM=noreply@yourdomain.com

# Optional: Telegram notifications
TELEGRAM_BOT_TOKEN=your-bot-token
```

## Notification Schedule

Default thresholds: 90, 60, 30, 14, 7, 3, 1 days before expiry

Daily notifications during final week before expiry.

Customize in Settings page.

## API

Domain Minder is primarily a web application. The following endpoints are available:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/dashboard` | User dashboard (auth required) |
| GET | `/domains` | List domains (auth required) |
| POST | `/domains` | Add domain (auth required) |
| POST | `/domains/:id/delete` | Delete domain (auth required) |
| GET | `/settings` | User settings (auth required) |
| POST | `/admin/check` | Trigger domain check (auth required) |

## Development

```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build binary
go build -o domain-minder ./cmd/server/

# Run with custom config
DB_PATH=/path/to/db SESSION_SECRET=dev ./domain-minder
```

## Deployment

### Docker

```bash
# Build image
docker build -t domain-minder .

# Run container
docker run -d \
  --name domain-minder \
  -p 9000:9000 \
  -v /path/to/data:/app/data \
  -e SESSION_SECRET=your-secret \
  domain-minder
```

### Systemd

Create `/etc/systemd/system/domain-minder.service`:

```ini
[Unit]
Description=Domain Minder - Domain Expiry Monitoring
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/domain-minder
ExecStart=/opt/domain-minder/domain-minder
Environment=DB_PATH=/opt/domain-minder/data/domain_minder.db
Environment=SESSION_SECRET=your-secret
Restart=always

[Install]
WantedBy=multi-user.target
```

### Nginx Proxy

```nginx
server {
    listen 80;
    server_name domain-minder.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl;
    server_name domain-minder.example.com;

    ssl_certificate /etc/letsencrypt/live/domain-minder.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/domain-minder.example.com/privkey.pem;

    location / {
        proxy_pass http://localhost:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Screenshots

### Dashboard
![Dashboard showing domains with progress bars](docs/dashboard.png)

### Add Domain
![Add domain form with WHOIS lookup](docs/add-domain.png)

### Settings
![Notification preferences and thresholds](docs/settings.png)

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and add tests
4. Run tests: `go test ./...`
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) file for details.
```

### 10.2 Create LICENSE File

Create `/home/adam/projects/domain-minder/LICENSE`:

```text
MIT License

Copyright (c) 2024 Your Name

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

### 10.3 Create Dockerfile

Create `/home/adam/projects/domain-minder/Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o domain-minder ./cmd/server/

# Run stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/domain-minder .
COPY --from=builder /app/.env.example .

# Create data directory
RUN mkdir -p data

# Environment variables (can be overridden)
ENV DB_PATH=/app/data/domain_minder.db
ENV PORT=9000

# Expose port
EXPOSE 9000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9000/health || exit 1

# Run application
ENTRYPOINT ["./domain-minder"]
```

### 10.4 Create Docker Compose

Create `/home/adam/projects/domain-minder/docker-compose.yml`:

```yaml
version: '3.8'

services:
  domain-minder:
    build: .
    container_name: domain-minder
    restart: unless-stopped
    ports:
      - "9000:9000"
    volumes:
      - ./data:/app/data
    environment:
      - DB_PATH=/app/data/domain_minder.db
      - PORT=9000
      - SESSION_SECRET=${SESSION_SECRET:-change-this-secret-in-production}
      - CHECK_INTERVAL=6h
      # Email settings (optional)
      - SMTP_HOST=${SMTP_HOST:-}
      - SMTP_PORT=${SMTP_PORT:-587}
      - SMTP_USER=${SMTP_USER:-}
      - SMTP_PASS=${SMTP_PASS:-}
      - SMTP_FROM=${SMTP_FROM:-noreply@localhost}
      # Telegram (optional)
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN:-}
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:9000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

volumes:
  data:
```

### 10.5 Create .env.example for Docker

Update `/home/adam/projects/domain-minder/.env.example`:

```bash
# Required
SESSION_SECRET=change-this-secret-in-production

# Optional: Email notifications
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@yourdomain.com

# Optional: Telegram notifications
TELEGRAM_BOT_TOKEN=
```

### 10.6 Create .gitignore

Create `/home/adam/projects/domain-minder/.gitignore`:

```gitignore
# Binary
domain-minder

# Data directory
data/*.db
data/*.db-journal

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Environment
.env
.env.local

# Coverage
coverage.out
coverage.html

# Logs
*.log

# Temp files
*.tmp
*.temp

# Go
go.work
go.work.sum
```

### 10.7 Create Development Guide

Create `/home/adam/projects/domain-minder/docs/DEVELOPMENT.md`:

```markdown
# Development Guide

## Getting Started

### Prerequisites

- Go 1.21+
- SQLite3
- Git

### Setup

```bash
# Clone repository
git clone https://github.com/CaffeinatedTech/domain-minder.git
cd domain-minder

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build binary
go build -o domain-minder ./cmd/server/

# Start development server
./domain-minder
```

### Project Structure

```
domain-minder/
├── cmd/server/           # Application entry point
├── internal/
│   ├── auth/            # Authentication utilities
│   ├── config/          # Configuration loading
│   ├── database/        # Database operations
│   ├── handlers/        # HTTP handlers
│   ├── middleware/      # Echo middleware
│   ├── models/          # Data models
│   ├── services/        # Business logic
│   │   ├── checker.go          # Background jobs
│   │   ├── notifications/      # Notification system
│   │   └── whois.go            # WHOIS service
│   └── templates/       # HTML templates
├── docs/                # Documentation
├── data/                # SQLite database (created at runtime)
├── .env                 # Environment variables
├── .env.example         # Environment template
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

### Adding New Features

1. Create handlers in `internal/handlers/`
2. Add routes in `cmd/server/main.go`
3. Create templates in `internal/templates/`
4. Add tests in respective `_test.go` files
5. Run `go mod tidy` if adding dependencies
6. Ensure all tests pass: `go test ./...`

### Database Migrations

For schema changes, update the `migrate()` function in `internal/database/database.go`:

```go
func migrate() error {
    // Add new tables or alter existing ones
    // Example:
    // _, err := DB.Exec(`ALTER TABLE users ADD COLUMN new_field TEXT`)
    // return err
}
```

### Code Style

- Follow Go conventions: `camelCase` for local variables, `PascalCase` for exports
- Use `echo.Context` for HTTP handling
- Return proper HTTP status codes
- Log errors with context
- Use context for cancellation

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/database/... -v
```

### Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Or use gofmt
gofmt -d .
```

## Release Process

1. Update version in `cmd/server/main.go` (if applicable)
2. Run tests: `go test ./...`
3. Run linter: `golangci-lint run`
4. Build binary: `go build -o domain-minder ./cmd/server/`
5. Test Docker build: `docker build -t domain-minder:test .`
6. Update CHANGELOG.md
7. Create git tag: `git tag v1.0.0`
8. Push to GitHub
```

### 10.8 Create CHANGELOG.md

Create `/home/adam/projects/domain-minder/CHANGELOG.md`:

```markdown
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
```

### 10.9 Create .dockerignore

Create `/home/adam/projects/domain-minder/.dockerignore`:

```gitignore
# Go
go.work
go.work.sum

# IDE
.idea
.vscode

# OS
.DS_Store
Thumbs.db

# Logs
*.log

# Temp
*.tmp
*.temp

# Coverage
coverage.out
coverage.html

# Git
.git
.gitignore

# Docs (optional, if you don't need them in image)
docs/

# Testing
*_test.go

# Docker
docker-compose.yml
Dockerfile
.dockerignore
```

### 10.10 Create Placeholder for Screenshots

Create `/home/adam/projects/domain-minder/docs/SCREENSHOTS.md`:

```markdown
# Screenshots

Add screenshots here by placing image files in this directory and updating the README.md references.

## Required Screenshots

1. **dashboard.png** - Main dashboard showing domains with progress bars
2. **add-domain.png** - Add domain form
3. **settings.png** - Settings page with notification preferences

## Image Specifications

- Format: PNG
- Width: 1200px (max)
- Quality: 80%

## Adding Screenshots

1. Take screenshots of the application
2. Save to `/home/adam/projects/domain-minder/docs/`
3. Update references in README.md
```

Create placeholder files:

```bash
# Create placeholder screenshots directory
mkdir -p /home/adam/projects/domain-minder/docs/screenshots

# Create placeholder files
touch /home/adam/projects/domain-minder/docs/screenshots/dashboard.png
touch /home/adam/projects/domain-minder/docs/screenshots/add-domain.png
touch /home/adam/projects/domain-minder/docs/screenshots/settings.png
```

## Deliverables

- [ ] `/home/adam/projects/domain-minder/README.md` - Updated with comprehensive documentation
- [ ] `/home/adam/projects/domain-minder/LICENSE` - MIT license
- [ ] `/home/adam/projects/domain-minder/Dockerfile` - Production-ready Docker image
- [ ] `/home/adam/projects/domain-minder/docker-compose.yml` - Development environment
- [ ] `/home/adam/projects/domain-minder/.env.example` - Environment template
- [ ] `/home/adam/projects/domain-minder/.gitignore` - Git ignore rules
- [ ] `/home/adam/projects/domain-minder/docs/DEVELOPMENT.md` - Development guide
- [ ] `/home/adam/projects/domain-minder/CHANGELOG.md` - Change log
- [ ] `/home/adam/projects/domain-minder/.dockerignore` - Docker ignore rules
- [ ] `/home/adam/projects/domain-minder/docs/screenshots/` - Placeholder directory for screenshots
- [ ] All existing tests still pass
- [ ] Docker image builds successfully

## Verification

```bash
cd /home/adam/projects/domain-minder

# Verify all files exist
ls -la Dockerfile docker-compose.yml .env.example .gitignore LICENSE README.md CHANGELOG.md

# Build Docker image
docker build -t domain-minder:test .

# Verify image built
docker images domain-minder:test

# Run tests one more time
go test ./...

# Check that binary builds
go build -o domain-minder ./cmd/server/
ls -la domain-minder

# Cleanup
rm -f domain-minder
```

## GitHub Setup (Manual Steps)

After completing this phase, perform these steps manually:

1. Create GitHub repository at https://github.com/new
2. Initialize git: `git init && git add . && git commit -m "Initial commit"`
3. Add remote: `git remote add origin https://github.com/CaffeinatedTech/domain-minder.git`
4. Push: `git push -u origin main`
5. Create release tag: `git tag v1.0.0 && git push origin v1.0.0`
6. Add screenshots to docs/screenshots/ and update README.md

## Project Complete

All phases are complete. Domain Minder is ready for release!
