# Phase 3: User Authentication

## What Should Already Exist

- `/home/adam/projects/domain-minder/go.mod` with dependencies
- `/home/adam/projects/domain-minder/internal/config/config.go` with Config struct
- `/home/adam/projects/domain-minder/cmd/server/main.go` with basic Echo server
- `/home/adam/projects/domain-minder/internal/database/database.go` with Init and migrate
- `/home/adam/projects/domain-minder/internal/database/users.go` with user CRUD functions
- `/home/adam/projects/domain-minder/internal/models/models.go` with User struct

## Context

This phase implements user authentication including registration, login, logout, session management using Echo's session middleware, and email verification flow. The email verification includes generating tokens, sending verification emails, and a dashboard banner for unverified emails.

## Agent Rules

1. Create files ONLY in `/home/adam/projects/domain-minder/internal/middleware/` and `/home/adam/projects/domain-minder/internal/handlers/`
2. Use Echo session middleware - NOT Gorilla sessions
3. Password hashing with bcrypt from `golang.org/x/crypto/bcrypt`
4. Session stored in secure HTTP-only cookies
5. All endpoints must handle errors and return appropriate HTTP responses
6. Email verification must include token generation, email sending, and verification endpoint
7. Dashboard must show banner when email is not verified
8. Do NOT create HTML templates yet - return simple HTML strings or JSON for now
9. Do NOT create domain handlers yet - only auth handlers
10. Use context for all database operations

## Tasks

### 3.1 Create Auth Middleware

Create `/home/adam/projects/domain-minder/internal/middleware/auth.go`:

```go
package middleware

import (
    "net/http"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/yourusername/domain-minder/internal/database"
    "github.com/yourusername/domain-minder/internal/models"
)

const (
    SessionUserKey = "user_id"
)

func SetupSessionMiddleware(e *echo.Echo, secret string) {
    config := middleware.SessionConfig{
        SessionName: "domain-minder-session",
        SessionStore: middleware.NewCookieStore([]byte(secret)),
        CookieHTTPOnly: true,
        CookieSecure: false, // Set to true in production with HTTPS
    }
    e.Use(middleware.SessionWithConfig(config))
}

func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        session, err := middleware.SessionManager.Get(c.Request(), "domain-minder-session")
        if err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "session error")
        }

        userID := session.Values[SessionUserKey]
        if userID == nil {
            return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
        }

        // Verify user still exists
        user, err := database.GetUserByID(c.Request().Context(), userID.(int))
        if err != nil || user == nil {
            return echo.NewHTTPError(http.StatusUnauthorized, "user not found")
        }

        c.Set("user", user)
        c.Set("user_id", user.ID)
        return next(c)
    }
}

func GetCurrentUser(c echo.Context) *models.User {
    user, ok := c.Get("user").(*models.User)
    if !ok {
        return nil
    }
    return user
}
```

### 3.2 Create Password Utilities

Create `/home/adam/projects/domain-minder/internal/auth/password.go`:

```go
package auth

import (
    "golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

### 3.3 Create Token Utilities

Create `/home/adam/projects/domain-minder/internal/auth/token.go`:

```go
package auth

import (
    "crypto/rand"
    "encoding/hex"
)

func GenerateVerificationToken() (string, error) {
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return hex.EncodeToString(bytes), nil
}
```

### 3.4 Create Email Utilities

Create `/home/adam/projects/domain-minder/internal/auth/email.go`:

```go
package auth

import (
    "bytes"
    "text/template"
    "strings"
    "net/smtp"
    "context"
    "fmt"
    "github.com/yourusername/domain-minder/internal/config"
)

var emailTemplates = map[string]string{
    "verification": `
<html>
<body>
<h2>Verify Your Email Address</h2>
<p>Thank you for registering with Domain Minder.</p>
<p>Click the link below to verify your email address:</p>
<p><a href="{{ .VerificationURL }}">Verify Email</a></p>
<p>This link will expire in 24 hours.</p>
<p>If you did not register for Domain Minder, please ignore this email.</p>
</body>
</html>
`,
    "notification": `
<html>
<body>
<h2>Domain Expiry Warning: {{ .DomainName }}</h2>
<p>Your domain <strong>{{ .DomainName }}</strong> registered with <strong>{{ .Registrar }}</strong></p>
<p>will expire in <strong>{{ .DaysRemaining }} days</strong> on {{ .ExpiryDate }}.</p>
<p>Log in to your <a href="{{ .DashboardURL }}">Domain Minder dashboard</a> for more details.</p>
</body>
</html>
`,
}

func SendVerificationEmail(ctx context.Context, cfg *config.Config, toEmail, verificationURL string) error {
    if cfg.SMTPConfig.Host == "" {
        return fmt.Errorf("SMTP not configured")
    }

    tmpl, ok := emailTemplates["verification"]
    if !ok {
        return fmt.Errorf("verification template not found")
    }

    var buf bytes.Buffer
    t, err := template.New("email").Parse(tmpl)
    if err != nil {
        return err
    }
    if err := t.Execute(&buf, map[string]string{"VerificationURL": verificationURL}); err != nil {
        return err
    }

    auth := smtp.PlainAuth("", cfg.SMTPConfig.Username, cfg.SMTPConfig.Password, cfg.SMTPConfig.Host)
    msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Verify your Domain Minder email\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
        cfg.SMTPConfig.From, toEmail, buf.String())

    return smtp.SendMail(
        fmt.Sprintf("%s:%d", cfg.SMTPConfig.Host, cfg.SMTPConfig.Port),
        auth,
        cfg.SMTPConfig.From,
        []string{toEmail},
        []byte(msg),
    )
}

func SendNotificationEmail(ctx context.Context, cfg *config.Config, toEmail, domainName, registrar, expiryDate string, daysRemaining int) error {
    if cfg.SMTPConfig.Host == "" {
        return fmt.Errorf("SMTP not configured")
    }

    tmpl, ok := emailTemplates["notification"]
    if !ok {
        return fmt.Errorf("notification template not found")
    }

    var buf bytes.Buffer
    t, err := template.New("email").Parse(tmpl)
    if err != nil {
        return err
    }
    if err := t.Execute(&buf, map[string]string{
        "DomainName":   domainName,
        "Registrar":    registrar,
        "ExpiryDate":   expiryDate,
        "DaysRemaining": fmt.Sprintf("%d", daysRemaining),
        "DashboardURL":  fmt.Sprintf("http://localhost:%d/dashboard", cfg.Port),
    }); err != nil {
        return err
    }

    auth := smtp.PlainAuth("", cfg.SMTPConfig.Username, cfg.SMTPConfig.Password, cfg.SMTPConfig.Host)
    msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Domain %s expires in %d days\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
        cfg.SMTPConfig.From, toEmail, domainName, daysRemaining, buf.String())

    return smtp.SendMail(
        fmt.Sprintf("%s:%d", cfg.SMTPConfig.Host, cfg.SMTPConfig.Port),
        auth,
        cfg.SMTPConfig.From,
        []string{toEmail},
        []byte(msg),
    )
}

func ParseEmailTemplate(name string, data interface{}) (string, error) {
    tmpl, ok := emailTemplates[name]
    if !ok {
        return "", fmt.Errorf("template not found: %s", name)
    }
    var buf bytes.Buffer
    t, err := template.New(name).Parse(tmpl)
    if err != nil {
        return "", err
    }
    if err := t.Execute(&buf, data); err != nil {
        return "", err
    }
    return strings.TrimSpace(buf.String()), nil
}
```

### 3.5 Create Auth Handlers

Create `/home/adam/projects/domain-minder/internal/handlers/auth.go`:

```go
package handlers

import (
    "net/http"
    "net/url"
    "context"
    "github.com/yourusername/domain-minder/internal/database"
    "github.com/yourusername/domain-minder/internal/models"
    "github.com/yourusername/domain-minder/internal/auth"
    "github.com/yourusername/domain-minder/internal/config"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

type AuthHandler struct {
    config *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
    return &AuthHandler{config: cfg}
}

func (h *AuthHandler) Register(c echo.Context) error {
    email := c.FormValue("email")
    password := c.FormValue("password")
    confirmPassword := c.FormValue("confirm_password")

    if email == "" || password == "" {
        return c.String(http.StatusBadRequest, "Email and password are required")
    }

    if password != confirmPassword {
        return c.String(http.StatusBadRequest, "Passwords do not match")
    }

    if len(password) < 8 {
        return c.String(http.StatusBadRequest, "Password must be at least 8 characters")
    }

    // Check if user exists
    existing, err := database.GetUserByEmail(c.Request().Context(), email)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "database error")
    }
    if existing != nil {
        return c.String(http.StatusBadRequest, "Email already registered")
    }

    // Hash password
    hash, err := auth.HashPassword(password)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
    }

    // Generate verification token
    token, err := auth.GenerateVerificationToken()
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
    }

    // Create user (email NOT verified initially)
    user := &models.User{
        Email:                  email,
        PasswordHash:           hash,
        EmailVerified:          false,
        EmailVerificationToken: &token,
        NotificationEmail:      true,
        NotificationTelegram:   false,
        NotificationThresholds: "[90, 60, 30, 14, 7, 3, 1]",
    }

    id, err := database.CreateUser(c.Request().Context(), user)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
    }
    user.ID = int(id)

    // Send verification email (don't block on error for now)
    verificationURL := h.buildVerificationURL(c, token)
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), h.config.CheckInterval)
        defer cancel()
        if err := auth.SendVerificationEmail(ctx, h.config, email, verificationURL); err != nil {
            // Log error but don't fail registration
            println("Failed to send verification email:", err.Error())
        }
    }()

    // Auto login
    session, _ := middleware.SessionManager.Get(c.Request(), "domain-minder-session")
    session.Values["user_id"] = user.ID
    session.Save(c.Request(), c.Response())

    return c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (h *AuthHandler) ShowRegister(c echo.Context) error {
    return c.String(http.StatusOK, `
    <html><body>
    <h1>Register</h1>
    <form method="POST" action="/register">
        <label>Email: <input type="email" name="email" required></label><br>
        <label>Password: <input type="password" name="password" required minlength="8"></label><br>
        <label>Confirm Password: <input type="password" name="confirm_password" required></label><br>
        <button type="submit">Register</button>
    </form>
    <a href="/login">Already have an account? Login</a>
    </body></html>
    `)
}

func (h *AuthHandler) Login(c echo.Context) error {
    email := c.FormValue("email")
    password := c.FormValue("password")

    if email == "" || password == "" {
        return c.String(http.StatusBadRequest, "Email and password are required")
    }

    user, err := database.GetUserByEmail(c.Request().Context(), email)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "database error")
    }
    if user == nil {
        return c.String(http.StatusUnauthorized, "Invalid email or password")
    }

    if !auth.CheckPassword(password, user.PasswordHash) {
        return c.String(http.StatusUnauthorized, "Invalid email or password")
    }

    session, _ := middleware.SessionManager.Get(c.Request(), "domain-minder-session")
    session.Values["user_id"] = user.ID
    session.Save(c.Request(), c.Response())

    return c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (h *AuthHandler) ShowLogin(c echo.Context) error {
    return c.String(http.StatusOK, `
    <html><body>
    <h1>Login</h1>
    <form method="POST" action="/login">
        <label>Email: <input type="email" name="email" required></label><br>
        <label>Password: <input type="password" name="password" required></label><br>
        <button type="submit">Login</button>
    </form>
    <a href="/register">Create an account</a>
    </body></html>
    `)
}

func (h *AuthHandler) Logout(c echo.Context) error {
    session, _ := middleware.SessionManager.Get(c.Request(), "domain-minder-session")
    session.Values["user_id"] = nil
    session.Save(c.Request(), c.Response())
    return c.Redirect(http.StatusSeeOther, "/login")
}

func (h *AuthHandler) VerifyEmail(c echo.Context) error {
    token := c.QueryParam("token")
    if token == "" {
        return c.String(http.StatusBadRequest, "Verification token required")
    }

    user, err := database.GetUserByVerificationToken(c.Request().Context(), token)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "database error")
    }
    if user == nil {
        return c.String(http.StatusBadRequest, "Invalid or expired verification token")
    }

    if err := database.UpdateUserVerification(c.Request().Context(), user.ID, true); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify email")
    }

    return c.String(http.StatusOK, `
    <html><body>
    <h1>Email Verified!</h1>
    <p>Your email has been successfully verified.</p>
    <a href="/dashboard">Go to Dashboard</a>
    </body></html>
    `)
}

func (h *AuthHandler) ResendVerification(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    if user.EmailVerified {
        return c.String(http.StatusBadRequest, "Email already verified")
    }

    // Generate new token
    token, err := auth.GenerateVerificationToken()
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
    }

    // Update user with new token
    user.EmailVerificationToken = &token
    if err := database.UpdateUser(c.Request().Context(), user); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to update user")
    }

    // Send verification email
    verificationURL := h.buildVerificationURL(c, token)
    if err := auth.SendVerificationEmail(c.Request().Context(), h.config, user.Email, verificationURL); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to send verification email")
    }

    return c.String(http.StatusOK, "Verification email sent")
}

func (h *AuthHandler) buildVerificationURL(c echo.Context, token string) string {
    scheme := "http"
    if c.Request().URL.Scheme != "" {
        scheme = c.Request().URL.Scheme
    }
    host := c.Request().Host
    if host == "" {
        host = "localhost:" + h.config.ServerAddr()
    }
    return scheme + "://" + host + "/verify?token=" + url.QueryEscape(token)
}
```

### 3.6 Create Settings Handler

Create `/home/adam/projects/domain-minder/internal/handlers/settings.go`:

```go
package handlers

import (
    "net/http"
    "github.com/yourusername/domain-minder/internal/database"
    "github.com/yourusername/domain-minder/internal/middleware"
    "github.com/labstack/echo/v4"
)

type SettingsHandler struct{}

func NewSettingsHandler() *SettingsHandler {
    return &SettingsHandler{}
}

func (h *SettingsHandler) ShowSettings(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    emailStatus := "Verified"
    if !user.EmailVerified {
        emailStatus = `<span style="color: red;">Not Verified</span> <a href="/verify/resend">[Resend]</a>`
    }

    return c.String(http.StatusOK, `
    <html><body>
    <h1>Settings</h1>
    <h2>Profile</h2>
    <p>Email: ` + user.Email + ` (` + emailStatus + `)</p>
    <h2>Notification Preferences</h2>
    <form method="POST" action="/settings/notifications">
        <label>
            <input type="checkbox" name="notification_email" ` + boolChecked(user.NotificationEmail) + `>
            Email notifications
        </label><br>
        <label>
            <input type="checkbox" name="notification_telegram" ` + boolChecked(user.NotificationTelegram) + `>
            Telegram notifications
        </label><br>
        <label>
            Telegram Chat ID: <input type="text" name="telegram_chat_id" value="` + nullString(user.TelegramChatID) + `">
        </label><br>
        <button type="submit">Save</button>
    </form>
    <h2>Notification Thresholds</h2>
    <form method="POST" action="/settings/thresholds">
        <label>
            Days before expiry (comma-separated):
            <input type="text" name="thresholds" value="` + user.NotificationThresholds + `">
        </label><br>
        <button type="submit">Save</button>
    </form>
    <a href="/dashboard">Back to Dashboard</a>
    </body></html>
    `)
}

func (h *SettingsHandler) UpdateNotifications(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    user.NotificationEmail = c.FormValue("notification_email") == "on"
    user.NotificationTelegram = c.FormValue("notification_telegram") == "on"

    tgChatID := c.FormValue("telegram_chat_id")
    if tgChatID != "" {
        user.TelegramChatID = &tgChatID
    } else {
        user.TelegramChatID = nil
    }

    if err := database.UpdateUser(c.Request().Context(), user); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to update settings")
    }

    return c.Redirect(http.StatusSeeOther, "/settings")
}

func (h *SettingsHandler) UpdateThresholds(c echo.Context) error {
    user := middleware.GetCurrentUser(c)
    if user == nil {
        return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
    }

    thresholds := c.FormValue("thresholds")
    user.NotificationThresholds = thresholds

    if err := database.UpdateUser(c.Request().Context(), user); err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "failed to update thresholds")
    }

    return c.Redirect(http.StatusSeeOther, "/settings")
}

func boolChecked(b bool) string {
    if b {
        return "checked"
    }
    return ""
}

func nullString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}
```

### 3.7 Update Main Server

Update `/home/adam/projects/domain-minder/cmd/server/main.go` to add auth routes and middleware:

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

    // Session middleware
    middleware.SetupSessionMiddleware(e, cfg.SessionSecret)

    // Health check
    e.GET("/health", func(c echo.Context) error {
        return c.String(http.StatusOK, "OK")
    })

    // Auth handlers
    authHandler := handlers.NewAuthHandler(cfg)
    settingsHandler := handlers.NewSettingsHandler()

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

    // Dashboard (protected)
    protected.GET("/dashboard", func(c echo.Context) error {
        user := middleware.GetCurrentUser(c)
        return c.String(http.StatusOK, `
        <html><body>
        <h1>Dashboard</h1>
        <p>Welcome, `+user.Email+`!</p>
        `+h.buildEmailVerificationBanner(user)+`
        <a href="/settings">Settings</a> |
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

func (h *AuthHandler) buildEmailVerificationBanner(user *models.User) string {
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

Note: The handler struct and methods need to be properly organized - the buildEmailVerificationBanner method should be on AuthHandler.

## Deliverables

- [ ] `/home/adam/projects/domain-minder/internal/middleware/auth.go` with session middleware and RequireAuth
- [ ] `/home/adam/projects/domain-minder/internal/auth/password.go` with HashPassword and CheckPassword
- [ ] `/home/adam/projects/domain-minder/internal/auth/token.go` with GenerateVerificationToken
- [ ] `/home/adam/projects/domain-minder/internal/auth/email.go` with email sending functions
- [ ] `/home/adam/projects/domain-minder/internal/handlers/auth.go` with registration, login, logout, verification
- [ ] `/home/adam/projects/domain-minder/internal/handlers/settings.go` with settings and notification preferences
- [ ] `/home/adam/projects/domain-minder/cmd/server/main.go` updated with auth routes and middleware
- [ ] Registration creates user with unverified email and sends verification email
- [ ] Login authenticates user and creates session
- [ ] Verification endpoint marks email as verified
- [ ] Dashboard shows warning banner when email not verified
- [ ] Resend verification generates new token and sends email

## Verification

```bash
cd /home/adam/projects/domain-minder

# Build
go build -o domain-minder ./cmd/server/

# Start server
./domain-minder &
sleep 2

# Test health
curl http://localhost:8080/health
# Expected: OK

# Test registration
curl -X POST http://localhost:8080/register \
  -d "email=test@example.com&password=test1234&confirm_password=test1234" \
  -c cookies.txt -L

# Check if verification email was "sent" (check logs)
# Note: Will fail if SMTP not configured, but registration should succeed

# Test login
curl -X POST http://localhost:8080/login \
  -d "email=test@example.com&password=test1234" \
  -c cookies.txt -b cookies.txt -L

# Test dashboard (should show email verification banner)
curl http://localhost:8080/dashboard -b cookies.txt

# Test settings page
curl http://localhost:8080/settings -b cookies.txt

# Cleanup
pkill -f domain-minder
rm -f domain_minder.db cookies.txt
```

## Next Phase

After completing verification, proceed to **Phase 4: Domain CRUD**.
