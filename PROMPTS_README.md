# Domain Minder - Implementation Prompts

This directory contains staged prompts for implementing Domain Minder. Each phase builds on the previous, with clear deliverables and constraints.

## Phase Order

Execute prompts in this order:

1. **[PHASE_01_PROJECT_SETUP.md](PHASE_01_PROJECT_SETUP.md)** - Initialize Go module, dependencies, directory structure, config system, basic Echo server
2. **[PHASE_02_DATABASE.md](PHASE_02_DATABASE.md)** - Database schema, models, and CRUD functions
3. **[PHASE_03_AUTH.md](PHASE_03_AUTH.md)** - User authentication, sessions, email verification flow
4. **[PHASE_04_DOMAIN_CRUD.md](PHASE_04_DOMAIN_CRUD.md)** - Domain management endpoints and handlers
5. **[PHASE_05_WHOIS.md](PHASE_05_WHOIS.md)** - WHOIS service for domain lookup and expiry parsing
6. **[PHASE_06_DASHBOARD.md](PHASE_06_DASHBOARD.md)** - Dashboard UI with HTMX and progress bars
7. **[PHASE_07_NOTIFICATIONS.md](PHASE_07_NOTIFICATIONS.md)** - Modular notification system (email/telegram)
8. **[PHASE_08_BACKGROUND_JOBS.md](PHASE_08_BACKGROUND_JOBS.md)** - Scheduler for automatic domain checking
9. **[PHASE_09_TESTING.md](PHASE_09_TESTING.md)** - Unit tests, handler tests, linting
10. **[PHASE_10_DOCS_RELEASE.md](PHASE_10_DOCS_RELEASE.md)** - Documentation, Docker, deployment

## General Agent Rules (Apply to All Phases)

- **ONE PHASE AT A TIME**: Only work on the current phase. Do not skip ahead.
- **USE EXISTING CODE**: Follow conventions from files already created in this phase or previous phases
- **NO EXTRA DEPENDENCIES**: Do not add libraries not listed in the prompt without approval
- **TEST AFTER IMPLEMENTATION**: Run `go test ./...` after completing tasks, fix any failures
- **COMMIT AFTER COMPLETION**: After verifying deliverables, commit with message "Phase X: [description]"
- **RUN LINTING**: Execute `golangci-lint run` or `gofmt -d` before committing
- **NO SPURIOUS CHANGES**: Do not modify unrelated files or add features not requested
- **USE ABSOLUTE PATHS**: All file paths must be absolute or relative to project root
- **KEEP HANDLERS THIN**: Business logic goes in services, handlers only handle HTTP
- **CONTEXT EVERYWHERE**: All long operations must accept context.Context
- **ERROR HANDLING**: Log errors with context, return appropriate HTTP errors
- **NO HARDCODED VALUES**: Use config/environment for all configurable values

## Context Required at Start of Each Phase

Each phase prompt specifies what should already exist. Verify this before starting. If something is missing, complete the previous phase first.

## Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package
go test ./internal/config/...
```

## Building

```bash
# Build binary
go build -o domain-minder ./cmd/server/

# Run server
./domain-minder
```
