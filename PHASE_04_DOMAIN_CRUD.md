# Phase 4: Domain CRUD

## What Should Already Exist

- `/home/adam/projects/domain-minder/cmd/server/main.go` with auth routes and middleware
- `/home/adam/projects/domain-minder/internal/middleware/auth.go` with RequireAuth middleware
- `/home/adam/projects/domain-minder/internal/database/domains.go` with domain CRUD functions
- `/home/adam/projects/domain-minder/internal/models/models.go` with Domain struct
- `/home/adam/projects/domain-minder/internal/handlers/auth.go` with auth handlers

## Context

This phase implements domain management operations: listing domains, adding new domains with WHOIS lookup, editing domain details, deleting domains, and triggering manual WHOIS checks. All endpoints are protected by authentication. HTML templates are still simple strings - full UI comes in Phase 6.

## Agent Rules

1. Create files ONLY in `/home/adam/projects/domain-minder/internal/handlers/`
2. All endpoints require authentication (use RequireAuth middleware)
3. Domain operations must verify the domain belongs to the current user
4. When adding a domain, perform WHOIS lookup to get expiry date and registrar
5. If WHOIS lookup fails, still allow manual entry of domain details
6. Return appropriate HTTP status codes (400 for bad input, 404 for not found, 403 for unauthorized)
7. Do NOT create HTML templates yet - return simple HTML strings or JSON
8. Do NOT implement background jobs yet - only manual domain operations
9. Use context for all database operations
10. Validate domain names with basic regex

## Tasks

### 4.1 Create Domain Handler

Create `/home/adam/projects/domain-minder/internal/handlers/domains.go`:

```go
package handlers

import (
    "net/http"
    "net/url"
    "regexp"
    "strconv"
    "time"
    "github.com/yourusername/domain-minder/internal/database"
    "github.com/yourusername/domain-minder/internal/middleware"
    "github.com/yourusername/domain-minder/internal/models"
    "github.com/yourusername/domain-minder/internal/services"
    "github.com/labstack/echo/v4"
)

type DomainHandler struct {
    whoisService *services.WHOISService
}

func NewDomainHandler(whoisService *services.WHOISService) *DomainHandler {
    return &DomainHandler{whoisService: whoisService}
}

func (h *DomainHandler) ListDomains(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    domains, err := database.GetDomainsByUserID(c.Request().Context(), user.ID)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domains")
    }

    var html string
    if len(domains) == 0 {
        html = `<p>No domains added yet.</p>`
    } else {
        html = `<table border="1"><tr><th>Domain</th><th>Registrar</th><th>Expiry</th><th>Days Left</th><th>Status</th><th>Actions</th></tr>`
        for _, d := range domains {
            daysLeft := int(time.Until(d.ExpiryDate).Hours() / 24)
            status := d.Status
            if daysLeft < 0 {
                status = "expired"
            }
            html += `<tr>
                <td>` + d.Name + `</td>
                <td>` + nullString(d.Registrar) + `</td>
                <td>` + d.ExpiryDate.Format("2006-01-02") + `</td>
                <td>` + strconv.Itoa(daysLeft) + `</td>
                <td>` + status + `</td>
                <td>
                    <form method="POST" action="/domains/` + strconv.Itoa(d.ID) + `/delete" style="display:inline;">
                        <button type="submit" onclick="return confirm('Delete this domain?')">Delete</button>
                    </form>
                </td>
            </tr>`
        }
        html += `</table>`
    }

    return c.String(http.StatusOK, `
    <html><body>
    <h1>Your Domains</h1>
    ` + html + `
    <h2>Add New Domain</h2>
    <form method="POST" action="/domains">
        <label>Domain Name: <input type="text" name="name" placeholder="example.com" required></label><br>
        <button type="submit">Add Domain</button>
    </form>
    <p><small>WHOIS lookup will be performed to find expiry date and registrar.</small></p>
    <a href="/dashboard">Back to Dashboard</a>
    </body></html>
    `)
}

func (h *DomainHandler) ShowAddDomain(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    return c.String(http.StatusOK, `
    <html><body>
    <h1>Add Domain</h1>
    <form method="POST" action="/domains">
        <label>Domain Name: <input type="text" name="name" required></label><br>
        <button type="submit">Add Domain</button>
    </form>
    <a href="/domains">Cancel</a>
    </body></html>
    `)
}

func (h *DomainHandler) AddDomain(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    domainName := c.FormValue("name")
    if !isValidDomain(domainName) {
        return echo.NewHTTPError(http.StatusBadRequest, "Invalid domain name")
    }

    // Perform WHOIS lookup
    result, err := h.whoisService.Lookup(domainName)
    if err != nil {
        // If WHOIS fails, still allow manual entry
        result = &services.WHOISResult{
            DomainName: domainName,
            Error:      err,
        }
    }

    registrar := result.Registrar
    if registrar == "" {
        registrar = "Unknown"
    }

    // Calculate expiry date
    expiryDate := result.ExpiryDate
    if expiryDate.IsZero() {
        // Default to 1 year from now if we couldn't get expiry
        expiryDate = time.Now().AddDate(1, 0, 0)
    }

    whoisRaw := ""
    if result.WHOISRaw != "" {
        whoisRaw = result.WHOISRaw
    }

    domain := &models.Domain{
        UserID:     user.ID,
        Name:       domainName,
        Registrar:  &registrar,
        ExpiryDate: expiryDate,
        WHOISRaw:   &whoisRaw,
        Status:     "active",
    }

    id, err := database.CreateDomain(c.Request().Context(), domain)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to create domain")
    }

    message := "Domain added successfully"
    if result.Error != nil {
        message += " (WHOIS lookup failed, default expiry date set)"
    }

    return c.Redirect(http.StatusSeeOther, "/domains?message="+url.QueryEscape(message))
}

func (h *DomainHandler) ShowEditDomain(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    domainID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid domain ID")
    }

    domain, err := database.GetDomainByID(c.Request().Context(), domainID)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domain")
    }
    if domain == nil {
        return echo.NewHTTPError(http.StatusNotFound, "domain not found")
    }
    if domain.UserID != user.ID {
        return echo.NewHTTPError(http.StatusForbidden, "not authorized")
    }

    return c.String(http.StatusOK, `
    <html><body>
    <h1>Edit Domain</h1>
    <form method="POST" action="/domains/`+strconv.Itoa(domain.ID)+`">
        <label>Domain Name: <input type="text" name="name" value="`+domain.Name+`" required></label><br>
        <label>Registrar: <input type="text" name="registrar" value="`+nullString(domain.Registrar)+`"></label><br>
        <label>Expiry Date: <input type="date" name="expiry_date" value="`+domain.ExpiryDate.Format("2006-01-02")+`" required></label><br>
        <label>Status:
            <select name="status">
                <option value="active"`+selected(domain.Status, "active")+`>Active</option>
                <option value="expired"`+selected(domain.Status, "expired")+`>Expired</option>
                <option value="pending"`+selected(domain.Status, "pending")+`>Pending</option>
            </select>
        </label><br>
        <label>Notes:<br><textarea name="notes">`+nullString(domain.Notes)+`</textarea></label><br>
        <button type="submit">Save</button>
    </form>
    <a href="/domains">Cancel</a>
    </body></html>
    `)
}

func (h *DomainHandler) UpdateDomain(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    domainID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid domain ID")
    }

    domain, err := database.GetDomainByID(c.Request().Context(), domainID)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domain")
    }
    if domain == nil {
        return echo.NewHTTPError(http.StatusNotFound, "domain not found")
    }
    if domain.UserID != user.ID {
        return echo.NewHTTPError(http.StatusForbidden, "not authorized")
    }

    domain.Name = c.FormValue("name")
    registrar := c.FormValue("registrar")
    domain.Registrar = &registrar
    expiryDateStr := c.FormValue("expiry_date")
    expiryDate, err := time.Parse("2006-01-02", expiryDateStr)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid expiry date format")
    }
    domain.ExpiryDate = expiryDate
    domain.Status = c.FormValue("status")
    notes := c.FormValue("notes")
    domain.Notes = &notes

    if err := database.UpdateDomain(c.Request().Context(), domain); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to update domain")
    }

    return c.Redirect(http.StatusSeeOther, "/domains")
}

func (h *DomainHandler) DeleteDomain(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    domainID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid domain ID")
    }

    domain, err := database.GetDomainByID(c.Request().Context(), domainID)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domain")
    }
    if domain == nil {
        return echo.NewHTTPError(http.StatusNotFound, "domain not found")
    }
    if domain.UserID != user.ID {
        return echo.NewHTTPError(http.StatusForbidden, "not authorized")
    }

    if err := database.DeleteDomain(c.Request().Context(), domainID); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete domain")
    }

    return c.Redirect(http.StatusSeeOther, "/domains")
}

func (h *DomainHandler) CheckDomain(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    domainID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid domain ID")
    }

    domain, err := database.GetDomainByID(c.Request().Context(), domainID)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domain")
    }
    if domain == nil {
        return echo.NewHTTPError(http.StatusNotFound, "domain not found")
    }
    if domain.UserID != user.ID {
        return echo.NewHTTPError(http.StatusForbidden, "not authorized")
    }

    // Perform WHOIS lookup
    result, err := h.whoisService.Lookup(domain.Name)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "WHOIS lookup failed: "+err.Error())
    }

    if !result.ExpiryDate.IsZero() {
        if err := database.UpdateDomainWHOIS(c.Request().Context(), domainID, result.ExpiryDate, result.Registrar, result.WHOISRaw); err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "failed to update domain")
        }
    }

    return c.Redirect(http.StatusSeeOther, "/domains?message="+url.QueryEscape("WHOIS check completed"))
}

func isValidDomain(name string) bool {
    // Basic domain validation
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]*\.[a-zA-Z]{2,}$`, name)
    return matched
}

func nullString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}

func selected(current, value string) string {
    if current == value {
        return " selected"
    }
    return ""
}
```

### 4.2 Create WHOIS Service Skeleton

Create `/home/adam/projects/domain-minder/internal/services/whois.go`:

```go
package services

import (
    "time"
)

type WHOISResult struct {
    DomainName string
    Registrar  string
    ExpiryDate time.Time
    WHOISRaw   string
    Error      error
}

type WHOISService struct{}

func NewWHOISService() *WHOISService {
    return &WHOISService{}
}

func (s *WHOISService) Lookup(domain string) (*WHOISResult, error) {
    // TODO: Implement in Phase 5
    return &WHOISResult{
        DomainName: domain,
        Error:      nil,
    }, nil
}
```

### 4.3 Update Main Server

Update `/home/adam/projects/domain-minder/cmd/server/main.go` to add domain routes:

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/yourusername/domain-minder/internal/config"
    "github.com/yourusername/domain-minder/internal/database"
    "github.com/yourusername/domain-minder/internal/handlers"
    "github.com/yourusername/domain-minder/internal/middleware"
    "github.com/yourusername/domain-minder/internal/services"
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

    e := echo.New()
    e.HideBanner = true

    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    e.Use(middleware.RequestID())

    middleware.SetupSessionMiddleware(e, cfg.SessionSecret)

    e.GET("/health", func(c echo.Context) error {
        return c.String(http.StatusOK, "OK")
    })

    authHandler := handlers.NewAuthHandler(cfg)
    settingsHandler := handlers.NewSettingsHandler()

    // WHOIS service (Phase 5 will implement actual lookup)
    whoisService := services.NewWHOISService()
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

    // Dashboard
    protected.GET("/dashboard", func(c echo.Context) error {
        user := middleware.GetCurrentUser(c)
        return c.String(http.StatusOK, `
        <html><body>
        <h1>Dashboard</h1>
        <p>Welcome, `+user.Email+`!</p>
        `+buildEmailVerificationBanner(user)+`
        <p><a href="/domains">Manage Domains</a></p>
        <p><a href="/settings">Settings</a></p>
        <form method="POST" action="/logout" style="display:inline;">
            <button type="submit">Logout</button>
        </form>
        </body></html>
        `)
    })

    log.Printf("Starting server on port %d", cfg.Port)
    if err := e.Start(cfg.ServerAddr()); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}

func buildEmailVerificationBanner(user *models.User) string {
    if !user.EmailVerified {
        return `
        <div style="background: #fff3cd; padding: 10px; margin: 10px 0; border: 1px solid #ffc107;">
            <strong>Warning:</strong> Your email is not verified.
            Email notifications are disabled until you verify.
            <a href="/verify/resend">Resend verification email</a>
        </div>
        `
    }
    return ""
}
```

Note: Add the missing import for models.User and move buildEmailVerificationBanner to be accessible.

## Deliverables

- [ ] `/home/adam/projects/domain-minder/internal/handlers/domains.go` with domain CRUD handlers
- [ ] `/home/adam/projects/domain-minder/internal/services/whois.go` with skeleton WHOIS service
- [ ] `/home/adam/projects/domain-minder/cmd/server/main.go` updated with domain routes
- [ ] GET `/domains` - Lists all user's domains in a table
- [ ] GET `/domains/new` - Shows add domain form
- [ ] POST `/domains` - Adds new domain with WHOIS lookup
- [ ] GET `/domains/:id` - Shows edit domain form
- [ ] POST `/domains/:id` - Updates domain
- [ ] POST `/domains/:id/delete` - Deletes domain
- [ ] POST `/domains/:id/check` - Triggers manual WHOIS check
- [ ] Domain validation rejects invalid domain names
- [ ] Authorization check ensures user can only access own domains

## Verification

```bash
cd /home/adam/projects/domain-minder

# Build
go build -o domain-minder ./cmd/server/

# Start server
./domain-minder &
sleep 2

# Login first (using existing test user or register new)
curl -X POST http://localhost:9000/login \
  -d "email=test@example.com&password=test1234" \
  -c cookies.txt -b cookies.txt -L

# Test listing domains (should be empty)
curl http://localhost:9000/domains -b cookies.txt

# Test adding a domain
curl -X POST http://localhost:9000/domains \
  -d "name=example.com" \
  -b cookies.txt -L

# Test listing domains again (should show example.com)
curl http://localhost:9000/domains -b cookies.txt

# Test editing a domain
curl http://localhost:9000/domains/1 -b cookies.txt

# Test deleting a domain
curl -X POST http://localhost:9000/domains/1/delete -b cookies.txt -L

# Cleanup
pkill -f domain-minder
rm -f domain_minder.db cookies.txt
```

## Next Phase

After completing verification, proceed to **Phase 5: WHOIS Service**.
