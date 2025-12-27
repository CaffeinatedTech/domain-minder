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
