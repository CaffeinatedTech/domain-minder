# Phase 8: Background Jobs

## What Should Already Exist

- `/home/adam/projects/domain-minder/internal/services/notifications/manager.go` with NotificationManager
- `/home/adam/projects/domain-minder/internal/services/whois.go` with WHOIS service
- `/home/adam/projects/domain-minder/internal/database/domains.go` with GetAllActiveDomains

## Context

This phase implements background job processing for automatic domain checking and notification sending. The scheduler runs on server startup, periodically checks all domains against WHOIS, updates expiry dates, and sends notifications when thresholds are reached. Includes graceful shutdown and manual trigger capability.

## Agent Rules

1. Create files ONLY in `/home/adam/projects/domain-minder/internal/services/checker.go`
2. Use `github.com/robfig/cron/v3` for scheduling (add to go.mod dependencies)
3. Background jobs must handle graceful shutdown on SIGTERM/SIGINT
4. All operations must use context for cancellation
5. Log all job executions and errors
6. Use the NotificationManager to send notifications
7. Do NOT create new handlers - only the background service
8. Add cron package to dependencies in go.mod
9. Include manual trigger endpoint in main.go
10. Job should run every 6 hours by default (configurable)

## Tasks

### 8.1 Add Cron Dependency

Update `/home/adam/projects/domain-minder/go.mod`:

```
require github.com/robfig/cron/v3 v3.0.1
```

Run: `go mod tidy`

### 8.2 Create Background Checker Service

Create `/home/adam/projects/domain-minder/internal/services/checker.go`:

```go
package services

import (
    "context"
    "log"
    "sync"
    "time"

    "github.com/robfig/cron/v3"

    "github.com/yourusername/domain-minder/internal/config"
    "github.com/yourusername/domain-minder/internal/database"
    "github.com/yourusername/domain-minder/internal/models"
    "github.com/yourusername/domain-minder/internal/services/notifications"
)

type Checker struct {
    cfg               *config.Config
    cron              *cron.Cron
    notificationMgr   *notifications.NotificationManager
    whoisService      *WHOISService
    running           bool
    runningMu         sync.Mutex
    stopChan          chan struct{}
}

func NewChecker(cfg *config.Config, notificationMgr *notifications.NotificationManager, whoisService *WHOISService) *Checker {
    return &Checker{
        cfg:             cfg,
        notificationMgr: notificationMgr,
        whoisService:    whoisService,
        stopChan:        make(chan struct{}),
    }
}

func (c *Checker) Start() error {
    c.runningMu.Lock()
    if c.running {
        c.runningMu.Unlock()
        return nil
    }
    c.running = true
    c.runningMu.Unlock()

    c.cron = cron.New(cron.WithSeconds())

    // Schedule domain check every 6 hours
    _, err := c.cron.AddFunc(c.cfg.CheckInterval.String(), c.CheckAllDomains)
    if err != nil {
        return err
    }

    c.cron.Start()

    log.Println("Background checker started")

    // Run initial check after short delay
    go func() {
        time.Sleep(10 * time.Second)
        c.CheckAllDomains()
    }()

    return nil
}

func (c *Checker) Stop() {
    c.runningMu.Lock()
    if !c.running {
        c.runningMu.Unlock()
        return
    }
    c.running = false
    c.runningMu.Unlock()

    if c.cron != nil {
        c.cron.Stop()
    }

    close(c.stopChan)
    log.Println("Background checker stopped")
}

func (c *Checker) CheckAllDomains() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()

    log.Println("Starting domain check...")

    domains, err := database.GetAllActiveDomains(ctx)
    if err != nil {
        log.Printf("Failed to get domains: %v", err)
        return
    }

    log.Printf("Checking %d domains", len(domains))

    checked := 0
    notificationsSent := 0
    errors := 0

    for _, domain := range domains {
        select {
        case <-ctx.Done():
            log.Println("Domain check cancelled due to timeout")
            return
        case <-c.stopChan:
            log.Println("Domain check stopped")
            return
        default:
        }

        // Get user for this domain
        user, err := database.GetUserByID(ctx, domain.UserID)
        if err != nil || user == nil {
            log.Printf("Failed to get user for domain %s: %v", domain.Name, err)
            errors++
            continue
        }

        // Update WHOIS information
        result, err := c.whoisService.Lookup(domain.Name)
        if err != nil {
            log.Printf("WHOIS lookup failed for %s: %v", domain.Name, err)
            errors++
        } else {
            if !result.ExpiryDate.IsZero() && !result.ExpiryDate.Equal(domain.ExpiryDate) {
                if err := database.UpdateDomainWHOIS(ctx, domain.ID, result.ExpiryDate, result.Registrar, result.WHOISRaw); err != nil {
                    log.Printf("Failed to update domain %s: %v", domain.Name, err)
                    errors++
                } else {
                    log.Printf("Updated %s expiry date to %s", domain.Name, result.ExpiryDate.Format("2006-01-02"))
                }
            }
            checked++
        }

        // Send notifications if needed
        logs, err := c.notificationMgr.CheckDomain(ctx, user, domain)
        if err != nil {
            log.Printf("Failed to check notifications for %s: %v", domain.Name, err)
            errors++
        } else {
            notificationsSent += len(logs)
        }
    }

    log.Printf("Domain check complete: %d checked, %d notifications sent, %d errors", checked, notificationsSent, errors)
}

func (c *Checker) RunOnce() {
    go c.CheckAllDomains()
}
```

### 8.3 Add Duration String Method

Add to `/home/adam/projects/domain-minder/internal/config/config.go`:

```go
func (c *Config) CheckIntervalString() string {
    if c.CheckInterval == 0 {
        return "0 */6 * * *"
    }
    hours := int(c.CheckInterval.Hours())
    return fmt.Sprintf("0 */%d * * *", hours)
}
```

### 8.4 Update Main Server

Update `/home/adam/projects/domain-minder/cmd/server/main.go` to include background checker:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/yourusername/domain-minder/internal/config"
    "github.com/yourusername/domain-minder/internal/database"
    "github.com/yourusername/domain-minder/internal/handlers"
    "github.com/yourusername/domain-minder/internal/middleware"
    "github.com/yourusername/domain-minder/internal/models"
    "github.com/yourusername/domain-minder/internal/services"
    "github.com/yourusername/domain-minder/internal/services/notifications"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    if err := os.MkdirAll("data", 0755); err != nil {
        log.Fatalf("Failed to create data directory: %v", err)
    }

    if err := database.Init(cfg); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer database.Close()

    // Initialize services
    whoisService := services.NewWHOISService()
    notificationMgr := notifications.NewNotificationManager(cfg)
    checker := services.NewChecker(cfg, notificationMgr, whoisService)

    // Start background checker
    if err := checker.Start(); err != nil {
        log.Fatalf("Failed to start checker: %v", err)
    }

    e := echo.New()
    e.HideBanner = true

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.RequestID())

    middleware.SetupSessionMiddleware(e, cfg.SessionSecret)

    e.GET("/health", func(c echo.Context) error {
        return c.String(http.StatusOK, "OK")
    })

    // Handlers
    authHandler := handlers.NewAuthHandler(cfg)
    settingsHandler := handlers.NewSettingsHandler()
    domainHandler := handlers.NewDomainHandler(whoisService)

    // Public routes
    e.GET("/register", authHandler.ShowRegister)
    e.POST("/register", authHandler.Register)
    e.GET("/login", authHandler.ShowLogin)
    e.POST("/login", authHandler.Login)
    e.GET("/verify", authHandler.VerifyEmail)

    // Protected routes
    protected := e.Group("")
    protected.Use(middleware.RequireAuth)
    protected.POST("/logout", authHandler.Logout)
    protected.GET("/verify/resend", authHandler.ResendVerification)
    protected.GET("/settings", settingsHandler.ShowSettings)
    protected.POST("/settings/notifications", settingsHandler.UpdateNotifications)
    protected.POST("/settings/thresholds", settingsHandler.UpdateThresholds)

    // Domain routes
    protected.GET("/domains", domainHandler.ListDomains)
    protected.GET("/domains/new", domainHandler.ShowAddDomain)
    protected.POST("/domains", domainHandler.AddDomain)
    protected.GET("/domains/:id", domainHandler.ShowEditDomain)
    protected.POST("/domains/:id", domainHandler.UpdateDomain)
    protected.POST("/domains/:id/delete", domainHandler.DeleteDomain)
    protected.POST("/domains/:id/check", domainHandler.CheckDomain)

    // Admin routes
    protected.POST("/admin/check", func(c echo.Context) error {
        checker.RunOnce()
        return c.String(http.StatusOK, "Check triggered")
    })

    // Dashboard
    protected.GET("/dashboard", func(c echo.Context) error {
        user := middleware.GetCurrentUser(c)
        // ... render dashboard template
        return c.String(http.StatusOK, "Dashboard")
    })

    // Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan

        log.Println("Shutting down...")
        checker.Stop()
        database.Close()
        os.Exit(0)
    }()

    log.Printf("Starting server on port %d", cfg.Port)
    if err := e.Start(cfg.ServerAddr()); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

## Deliverables

- [ ] `/home/adam/projects/domain-minder/internal/services/checker.go` with Checker service
- [ ] Background scheduler starts on server startup
- [ ] Domain check runs every 6 hours (configurable)
- [ ] WHOIS information updated for each domain
- [ ] Notifications sent based on user's threshold settings
- [ ] Graceful shutdown on SIGINT/SIGTERM
- [ ] Manual trigger endpoint at POST `/admin/check`
- [ ] Initial check runs 10 seconds after startup
- [ ] Job execution logged

## Verification

```bash
cd /home/adam/projects/domain-minder

# Add cron dependency
go get github.com/robfig/cron/v3@v3.0.1

# Build
go build -o domain-minder ./cmd/server/

# Start server
./domain-minder &
sleep 2

# Check logs for background checker startup
tail -f /tmp/domain-minder.log 2>/dev/null || journalctl -u domain-minder 2>/dev/null || echo "Check server output"

# Trigger manual check
curl -X POST http://localhost:9000/admin/check \
  -b cookies.txt -L

# Wait for check to complete (check logs)
sleep 5

# Test graceful shutdown
kill -TERM $(pgrep domain-minder)

# Verify clean shutdown in logs
```

## Next Phase

After completing verification, proceed to **Phase 9: Testing**.
