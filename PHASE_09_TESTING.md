# Phase 9: Testing

## What Should Already Exist

- All Phase 1-8 deliverables complete
- Server builds and runs without errors
- All handlers implemented

## Context

This phase implements comprehensive tests including unit tests for utilities, handler tests with mocked dependencies, and integration tests. Target 80%+ coverage on core functionality. Configure linting and fix all issues.

## Agent Rules

1. Create test files with `_test.go` suffix
2. Use standard `testing` package
3. Use `httptest` for handler tests
4. Mock database layer for handler tests
5. Run `go test ./...` after creating tests
6. Fix any test failures before proceeding
7. Run `golangci-lint run` or `gofmt -d` and fix issues
8. Do NOT modify production code to make tests pass - fix tests instead
9. Aim for 80%+ coverage on: config, database, handlers, services
10. Create integration test that exercises full flow

## Tasks

### 9.1 Create Config Tests

Create `/home/adam/projects/domain-minder/internal/config/config_test.go`:

```go
package config

import (
    "os"
    "testing"
)

func TestLoad(t *testing.T) {
    // Clean environment
    os.Clearenv()

    // Set test values
    os.Setenv("DB_PATH", "/test/db.sqlite")
    os.Setenv("PORT", "9999")
    os.Setenv("SESSION_SECRET", "test-secret")
    os.Setenv("CHECK_INTERVAL", "1h")

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load() error = %v", err)
    }

    if cfg.DBPath != "/test/db.sqlite" {
        t.Errorf("DBPath = %v, want /test/db.sqlite", cfg.DBPath)
    }

    if cfg.Port != 9999 {
        t.Errorf("Port = %v, want 9999", cfg.Port)
    }

    if cfg.SessionSecret != "test-secret" {
        t.Errorf("SessionSecret = %v, want test-secret", cfg.SessionSecret)
    }

    if cfg.CheckInterval.Hours() != 1 {
        t.Errorf("CheckInterval = %v, want 1h", cfg.CheckInterval)
    }

    // Cleanup
    os.Unsetenv("DB_PATH")
    os.Unsetenv("PORT")
    os.Unsetenv("SESSION_SECRET")
    os.Unsetenv("CHECK_INTERVAL")
}

func TestLoadDefaults(t *testing.T) {
    os.Clearenv()

    cfg, err := Load()
    if err != nil {
        t.Fatalf("Load() error = %v", err)
    }

    if cfg.DBPath != "data/domain_minder.db" {
        t.Errorf("Default DBPath = %v, want data/domain_minder.db", cfg.DBPath)
    }

    if cfg.Port != 9000 {
        t.Errorf("Default Port = %v, want 9000", cfg.Port)
    }
}
```

### 9.2 Create Auth Tests

Create `/home/adam/projects/domain-minder/internal/auth/password_test.go`:

```go
package auth

import (
    "testing"
)

func TestHashPassword(t *testing.T) {
    password := "testpassword123"

    hash, err := HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword() error = %v", err)
    }

    if hash == password {
        t.Error("Hash should not equal password")
    }

    if len(hash) < 10 {
        t.Error("Hash too short")
    }
}

func TestCheckPassword(t *testing.T) {
    password := "testpassword123"

    hash, err := HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword() error = %v", err)
    }

    if !CheckPassword(password, hash) {
        t.Error("CheckPassword() should return true for correct password")
    }

    if CheckPassword("wrongpassword", hash) {
        t.Error("CheckPassword() should return false for wrong password")
    }
}

func TestHashPasswordUnique(t *testing.T) {
    password := "testpassword123"

    hash1, _ := HashPassword(password)
    hash2, _ := HashPassword(password)

    // Same password should produce different hashes due to salt
    if hash1 == hash2 {
        t.Error("Same password should produce different hashes")
    }

    // But both should validate
    if !CheckPassword(password, hash1) {
        t.Error("First hash should validate")
    }
    if !CheckPassword(password, hash2) {
        t.Error("Second hash should validate")
    }
}
```

### 9.3 Create Token Tests

Create `/home/adam/projects/domain-minder/internal/auth/token_test.go`:

```go
package auth

import (
    "testing"
)

func TestGenerateVerificationToken(t *testing.T) {
    token1, err := GenerateVerificationToken()
    if err != nil {
        t.Fatalf("GenerateVerificationToken() error = %v", err)
    }

    if len(token1) != 64 { // 32 bytes = 64 hex characters
        t.Errorf("Token length = %d, want 64", len(token1))
    }

    token2, err := GenerateVerificationToken()
    if err != nil {
        t.Fatalf("GenerateVerificationToken() error = %v", err)
    }

    if token1 == token2 {
        t.Error("Tokens should be unique")
    }
}

func TestGenerateVerificationTokenFormat(t *testing.T) {
    token, _ := GenerateVerificationToken()

    for _, c := range token {
        if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
            t.Errorf("Token contains invalid character: %c", c)
        }
    }
}
```

### 9.4 Create Handler Tests

Create `/home/adam/projects/domain-minder/internal/handlers/auth_test.go`:

```go
package handlers

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "github.com/labstack/echo/v4"
)

func TestShowRegister(t *testing.T) {
    e := echo.New()
    req := httptest.NewRequest(http.MethodGet, "/register", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    h := &AuthHandler{}
    if err := h.ShowRegister(c); err != nil {
        t.Fatalf("ShowRegister() error = %v", err)
    }

    if rec.Code != http.StatusOK {
        t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
    }

    if !strings.Contains(rec.Body.String(), "Register") {
        t.Error("Response should contain 'Register'")
    }
}

func TestShowLogin(t *testing.T) {
    e := echo.New()
    req := httptest.NewRequest(http.MethodGet, "/login", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    h := &AuthHandler{}
    if err := h.ShowLogin(c); err != nil {
        t.Fatalf("ShowLogin() error = %v", err)
    }

    if rec.Code != http.StatusOK {
        t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
    }

    if !strings.Contains(rec.Body.String(), "Login") {
        t.Error("Response should contain 'Login'")
    }
}

func TestRegisterValidation(t *testing.T) {
    e := echo.New()

    tests := []struct {
        name       string
        email      string
        password   string
        confirm    string
        wantStatus int
    }{
        {
            name:       "empty email",
            email:      "",
            password:   "password123",
            confirm:    "password123",
            wantStatus: http.StatusBadRequest,
        },
        {
            name:       "empty password",
            email:      "test@example.com",
            password:   "",
            confirm:    "",
            wantStatus: http.StatusBadRequest,
        },
        {
            name:       "password mismatch",
            email:      "test@example.com",
            password:   "password123",
            confirm:    "password456",
            wantStatus: http.StatusBadRequest,
        },
        {
            name:       "password too short",
            email:      "test@example.com",
            password:   "short",
            confirm:    "short",
            wantStatus: http.StatusBadRequest,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            form := "email=" + tt.email + "&password=" + tt.password + "&confirm_password=" + tt.confirm
            req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form))
            req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
            rec := httptest.NewRecorder()
            c := e.NewContext(req, rec)

            h := &AuthHandler{}
            h.Register(c)

            if rec.Code != tt.wantStatus {
                t.Errorf("Status code = %d, want %d", rec.Code, tt.wantStatus)
            }
        })
    }
}
```

### 9.5 Create Database Tests

Create `/home/adam/projects/domain-minder/internal/database/database_test.go`:

```go
package database

import (
    "context"
    "os"
    "testing"
    "github.com/CaffeinatedTech/domain-minder/internal/config"
)

func setupTestDB(t *testing.T) {
    os.Remove("/tmp/domain_minder_test.db")

    cfg := &config.Config{
        DBPath: "/tmp/domain_minder_test.db",
    }

    if err := Init(cfg); err != nil {
        t.Fatalf("Failed to initialize test database: %v", err)
    }
}

func teardownTestDB() {
    os.Remove("/tmp/domain_minder_test.db")
}

func TestDatabaseInit(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB()

    if DB == nil {
        t.Error("Database should be initialized")
    }

    if err := DB.Ping(); err != nil {
        t.Errorf("Ping() error = %v", err)
    }
}

func TestUserCRUD(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB()

    ctx := context.Background()

    // Create user
    user := &User{
        Email:              "test@example.com",
        PasswordHash:       "hash",
        EmailVerified:      false,
        NotificationEmail:  true,
        NotificationThresholds: "[90, 60, 30]",
    }

    id, err := CreateUser(ctx, user)
    if err != nil {
        t.Fatalf("CreateUser() error = %v", err)
    }

    user.ID = int(id)

    // Get user by ID
    fetched, err := GetUserByID(ctx, int(id))
    if err != nil {
        t.Fatalf("GetUserByID() error = %v", err)
    }

    if fetched.Email != user.Email {
        t.Errorf("Email = %v, want %v", fetched.Email, user.Email)
    }

    // Get user by email
    byEmail, err := GetUserByEmail(ctx, "test@example.com")
    if err != nil {
        t.Fatalf("GetUserByEmail() error = %v", err)
    }

    if byEmail.ID != user.ID {
        t.Errorf("User ID = %v, want %v", byEmail.ID, user.ID)
    }

    // Clean up
    DB.Exec("DELETE FROM users WHERE id = ?", id)
}

func TestDomainCRUD(t *testing.T) {
    setupTestDB(t)
    defer teardownTestDB()

    ctx := context.Background()

    // Create user first
    user := &User{
        Email:              "domain-test@example.com",
        PasswordHash:       "hash",
        NotificationThresholds: "[90, 60, 30]",
    }
    userID, _ := CreateUser(ctx, user)

    // Create domain
    registrar := "Test Registrar"
    domain := &Domain{
        UserID:     int(userID),
        Name:       "example.com",
        Registrar:  &registrar,
        ExpiryDate: time.Now().AddDate(1, 0, 0),
        Status:     "active",
    }

    domainID, err := CreateDomain(ctx, domain)
    if err != nil {
        t.Fatalf("CreateDomain() error = %v", err)
    }

    domain.ID = int(domainID)

    // Get domain by ID
    fetched, err := GetDomainByID(ctx, int(domainID))
    if err != nil {
        t.Fatalf("GetDomainByID() error = %v", err)
    }

    if fetched.Name != domain.Name {
        t.Errorf("Name = %v, want %v", fetched.Name, domain.Name)
    }

    // Get domains by user ID
    domains, err := GetDomainsByUserID(ctx, int(userID))
    if err != nil {
        t.Fatalf("GetDomainsByUserID() error = %v", err)
    }

    if len(domains) != 1 {
        t.Errorf("Number of domains = %d, want 1", len(domains))
    }

    // Clean up
    DB.Exec("DELETE FROM domains WHERE id = ?", domainID)
    DB.Exec("DELETE FROM users WHERE id = ?", userID)
}
```

### 9.6 Create Integration Test

Create `/home/adam/projects/domain-minder/internal/integration_test.go`:

```go
package integration

import (
    "context"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"
    "time"

    "github.com/CaffeinatedTech/domain-minder/internal/auth"
    "github.com/CaffeinatedTech/domain-minder/internal/config"
    "github.com/CaffeinatedTech/domain-minder/internal/database"
    "github.com/CaffeinatedTech/domain-minder/internal/handlers"
    "github.com/CaffeinatedTech/domain-minder/internal/models"
    "github.com/CaffeinatedTech/domain-minder/internal/services"
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
)

func setupTestEnvironment(t *testing.T) (*echo.Echo, *config.Config) {
    os.Remove("/tmp/domain_minder_integration.db")

    cfg := &config.Config{
        DBPath:        "/tmp/domain_minder_integration.db",
        Port:          18080,
        SessionSecret: "test-secret-integration",
        CheckInterval: 1 * time.Hour,
    }

    if err := database.Init(cfg); err != nil {
        t.Fatalf("Failed to initialize database: %v", err)
    }

    e := echo.New()
    middleware.SetupSessionMiddleware(e, cfg.SessionSecret)

    whoisService := services.NewWHOISService()
    domainHandler := handlers.NewDomainHandler(whoisService)
    authHandler := handlers.NewAuthHandler(cfg)

    e.GET("/register", authHandler.ShowRegister)
    e.POST("/register", authHandler.Register)
    e.GET("/login", authHandler.ShowLogin)
    e.POST("/login", authHandler.Login)

    protected := e.Group("")
    protected.Use(middleware.RequireAuth)
    protected.GET("/domains", domainHandler.ListDomains)
    protected.POST("/domains", domainHandler.AddDomain)

    return e, cfg
}

func teardownTestEnvironment() {
    database.Close()
    os.Remove("/tmp/domain_minder_integration.db")
}

func TestFullUserFlow(t *testing.T) {
    e, cfg := setupTestEnvironment(t)
    defer teardownTestEnvironment()

    // 1. Register
    t.Run("Register", func(t *testing.T) {
        form := "email=flowtest@example.com&password=testpassword123&confirm_password=testpassword123"
        req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        rec := httptest.NewRecorder()
        e.ServeHTTP(rec, req)

        if rec.Code != http.StatusSeeOther {
            t.Errorf("Register status = %d, want %d", rec.Code, http.StatusSeeOther)
        }
    })

    // 2. Login
    t.Run("Login", func(t *testing.T) {
        jar := &http.CookieJar{}
        client := &http.Client{Jar: jar}

        form := "email=flowtest@example.com&password=testpassword123"
        req, _ := http.NewRequest(http.MethodPost, "/login", strings.NewReader(form))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        resp, err := client.Do(req)
        if err != nil {
            t.Fatalf("Login request failed: %v", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            t.Errorf("Login status = %d, want %d", resp.StatusCode, http.StatusOK)
        }
    })

    // 3. Add Domain
    t.Run("AddDomain", func(t *testing.T) {
        jar := &http.CookieJar{}
        client := &http.Client{Jar: jar}

        // First login to get cookies
        form := "email=flowtest@example.com&password=testpassword123"
        req, _ := http.NewRequest(http.MethodPost, "/login", strings.NewReader(form))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        client.Do(req)

        // Then add domain
        domainForm := "name=integration-test.com"
        req, _ = http.NewRequest(http.MethodPost, "/domains", strings.NewReader(domainForm))
        req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
        resp, err := client.Do(req)
        if err != nil {
            t.Fatalf("Add domain request failed: %v", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            t.Errorf("Add domain status = %d, want %d", resp.StatusCode, http.StatusOK)
        }
    })

    // 4. Verify Domain in Database
    t.Run("VerifyDomainInDB", func(t *testing.T) {
        user, _ := database.GetUserByEmail(context.Background(), "flowtest@example.com")
        if user == nil {
            t.Fatal("User not found")
        }

        domains, err := database.GetDomainsByUserID(context.Background(), user.ID)
        if err != nil {
            t.Fatalf("GetDomainsByUserID() error = %v", err)
        }

        if len(domains) == 0 {
            t.Error("Expected at least one domain")
        }

        found := false
        for _, d := range domains {
            if d.Name == "integration-test.com" {
                found = true
                break
            }
        }

        if !found {
            t.Error("Domain 'integration-test.com' not found")
        }
    })
}

func TestPasswordHashing(t *testing.T) {
    password := "mysecurepassword123"

    hash, err := auth.HashPassword(password)
    if err != nil {
        t.Fatalf("HashPassword() error = %v", err)
    }

    if !auth.CheckPassword(password, hash) {
        t.Error("CheckPassword() should return true for correct password")
    }

    if auth.CheckPassword("wrongpassword", hash) {
        t.Error("CheckPassword() should return false for wrong password")
    }
}

func TestVerificationToken(t *testing.T)    token1, err := auth.GenerateVerificationToken()
    if err != nil {
        t.Fatalf("GenerateVerificationToken() error = %v", err)
    }

    if len(token1) != 64 {
        t.Errorf("Token length = %d, want 64", len(token1))
    }

    token2, err := auth.GenerateVerificationToken()
    if err != nil {
        t.Fatalf("GenerateVerificationToken() error = %v", err)
    }

    if token1 == token2 {
        t.Error("Tokens should be unique")
    }
}
```

### 9.7 Run Tests and Fix Issues

```bash
cd /home/adam/projects/domain-minder

# Run all tests
go test ./... -v -cover

# Check coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Run specific package
go test ./internal/config/... -v
go test ./internal/auth/... -v
go test ./internal/database/... -v
go test ./internal/handlers/... -v

# Install and run linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run

# Or use gofmt
gofmt -d .
```

### 9.8 Add Test to Main Test Suite

Create `/home/adam/projects/domain-minder/cmd/server/main_test.go`:

```go
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/labstack/echo/v4"
)

func TestHealthEndpoint(t *testing.T) {
    e := echo.New()
    req := httptest.NewRequest(http.MethodGet, "/health", nil)
    rec := httptest.NewRecorder()
    c := e.NewContext(req, rec)

    // Mock handler from main
    handler := func(c echo.Context) error {
        return c.String(http.StatusOK, "OK")
    }

    if err := handler(c); err != nil {
        t.Fatalf("Handler error = %v", err)
    }

    if rec.Code != http.StatusOK {
        t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
    }

    if rec.Body.String() != "OK" {
        t.Errorf("Body = %s, want OK", rec.Body.String())
    }
}
```

## Deliverables

- [ ] `/home/adam/projects/domain-minder/internal/config/config_test.go`
- [ ] `/home/adam/projects/domain-minder/internal/auth/password_test.go`
- [ ] `/home/adam/projectsm/internal/auth/token_test.go`
- [ ] `/home/adam/projects/domain-minder/internal/handlers/auth_test.go`
- [ ] `/home/adam/projects/domain-minder/internal/database/database_test.go`
- [ ] `/home/adam/projects/domain-minder/internal/integration_test.go`
- [ ] `/home/adam/projects/domain-minder/cmd/server/main_test.go`
- [ ] 80%+ coverage on config, auth, database, handlers
- [ ] All tests pass
- [ ] Linting passes with no errors

## Verification

```bash
cd /home/adam/projects/domain-minder

# Run all tests
go test ./... -v

# Check coverage
go test ./... -cover

# Output should show:
# ok    github.com/CaffeinatedTech/domain-minder/internal/config    0.xxxs  100.0%
# ok    github.com/CaffeinatedTech/domain-minder/internal/auth      0.xxxs  100.0%
# ok    github.com/CaffeinatedTech/domain-minder/internal/database  0.xxxs  85.0%
# ...

# Linting
golangci-lint run
# Or
gofmt -d .
```

## Next Phase

After completing verification, proceed to **Phase 10: Documentation & Release**.
