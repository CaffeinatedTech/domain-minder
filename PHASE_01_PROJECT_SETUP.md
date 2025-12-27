# Phase 1: Project Setup

## What Should Already Exist

Nothing. This is the first phase.

## Context

This phase establishes the foundation of the Domain Minder project. You will create the Go module, directory structure, configuration system, and a basic Echo server that responds to a health check endpoint.

## Agent Rules

1. Create files ONLY as specified below - do not create handlers, models, or services yet
2. Use the exact directory structure shown
3. Add ONLY the dependencies listed
4. Configuration must support environment variables and `.env` file loading
5. The server must start without errors and respond to `/health`

## Tasks

### 1.1 Initialize Go Module

Create `go.mod` in `/home/adam/projects/domain-minder/` with:

```
module github.com/yourusername/domain-minder

go 1.21

require (
    github.com/labstack/echo/v4 v4.11.4
    github.com/mattn/go-sqlite3 v1.14.22
    github.com/labstack/echo/v4/middleware/session v4.11.4
    golang.org/x/crypto v0.17.0
)
```

### 1.2 Download Dependencies

Run: `go mod tidy`

### 1.3 Create Directory Structure

Create these directories:

```
/home/adam/projects/domain-minder/cmd/server
/home/adam/projects/domain-minder/internal/config
/home/adam/projects/domain-minder/internal/database
/home/adam/projects/domain-minder/internal/models
/home/adam/projects/domain-minder/internal/handlers
/home/adam/projects/domain-minder/internal/services/notifications
/home/adam/projects/domain-minder/internal/middleware
/home/adam/projects/domain-minder/internal/templates
/home/adam/projects/domain-minder/migrations
/home/adam/projects/domain-minder/data
```

### 1.4 Create Configuration System

Create `/home/adam/projects/domain-minder/internal/config/config.go`:

```go
package config

import (
    "os"
    "strconv"
    "time"

    "github.com/joho/godotenv"
)

type Config struct {
    DBPath            string
    Port              int
    SessionSecret     string
    SMTPConfig        SMTPConfig
    TelegramConfig    TelegramConfig
    CheckInterval     time.Duration
}

type SMTPConfig struct {
    Host     string
    Port     int
    Username string
    Password string
    From     string
}

type TelegramConfig struct {
    BotToken string
}

func Load() (*Config, error) {
    godotenv.Load()

    cfg := &Config{
        DBPath:        getEnv("DB_PATH", "data/domain_minder.db"),
        Port:          getEnvInt("PORT", 9000),
        SessionSecret: getEnv("SESSION_SECRET", "change-this-in-production"),
        CheckInterval: getEnvDuration("CHECK_INTERVAL", 6 * time.Hour),
        SMTPConfig: SMTPConfig{
            Host:     getEnv("SMTP_HOST", ""),
            Port:     getEnvInt("SMTP_PORT", 587),
            Username: getEnv("SMTP_USER", ""),
            Password: getEnv("SMTP_PASS", ""),
            From:     getEnv("SMTP_FROM", "noreply@domain-minder.local"),
        },
        TelegramConfig: TelegramConfig{
            BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
        },
    }

    return cfg, nil
}

func getEnv(key, defaultValue string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if v := os.Getenv(key); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
    if v := os.Getenv(key); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            return d
        }
    }
    return defaultValue
}
```

### 1.5 Create Main Entry Point

Create `/home/adam/projects/domain-minder/cmd/server/main.go`:

```go
package main

import (
    "log"
    "os"

    "github.com/yourusername/domain-minder/internal/config"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Ensure data directory exists
    if err := os.MkdirAll("data", 0755); err != nil {
        log.Fatalf("Failed to create data directory: %v", err)
    }

    e := echo.New()
    e.HideBanner = true

    // Middleware
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.RequestID())

    // Health check endpoint
    e.GET("/health", func(c echo.Context) error {
        return c.String(200, "OK")
    })

    log.Printf("Starting server on port %d", cfg.Port)
    if err := e.Start(":" + cfg.ServerAddr()); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

Note: Add `ServerAddr()` method to Config that returns `fmt.Sprintf(":%d", cfg.Port)`.

### 1.6 Create .env.example

Create `/home/adam/projects/domain-minder/.env.example`:

```
DB_PATH=data/domain_minder.db
PORT=9000
SESSION_SECRET=your-super-secret-session-key-change-in-production
CHECK_INTERVAL=6h

# SMTP Configuration (for email notifications)
SMTP_HOST=
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
SMTP_FROM=noreply@yourdomain.com

# Telegram Configuration
TELEGRAM_BOT_TOKEN=
```

### 1.7 Create data Directory

Create empty `/home/adam/projects/domain-minder/data/` directory.

## Deliverables

- [ ] `go.mod` created with correct dependencies
- [ ] `go mod tidy` executed successfully
- [ ] All directories created
- [ ] `/home/adam/projects/domain-minder/internal/config/config.go` with Config struct and Load function
- [ ] `/home/adam/projects/domain-minder/cmd/server/main.go` with basic Echo server
- [ ] `/home/adam/projects/domain-minder/.env.example` configuration template
- [ ] Server starts without errors
- [ ] `curl http://localhost:9000/health` returns "OK"

## Verification

```bash
cd /home/adam/projects/domain-minder

# Build the server
go build -o domain-minder ./cmd/server/

# Start in background
./domain-minder &
sleep 2

# Test health endpoint
curl http://localhost:9000/health
# Expected: OK

# Kill server
pkill -f domain-minder

# Run tests (should pass without failures)
go test ./...
```

## Next Phase

After completing verification, proceed to **Phase 2: Database**.
