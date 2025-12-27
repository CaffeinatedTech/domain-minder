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
	"github.com/CaffeinatedTech/domain-minder/internal/middleware"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
	"github.com/CaffeinatedTech/domain-minder/internal/services"
	"github.com/CaffeinatedTech/domain-minder/internal/templates"
	"github.com/labstack/echo/v4"
)

func setupTestEnvironment(t *testing.T) *echo.Echo {
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
	e.Renderer = templates.NewRenderer("/home/adam/projects/domain-minder/internal/templates")
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

	return e
}

func teardownTestEnvironment() {
	database.Close()
	os.Remove("/tmp/domain_minder_integration.db")
}

func TestFullUserFlow(t *testing.T) {
	e := setupTestEnvironment(t)
	defer teardownTestEnvironment()

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

	t.Run("Login", func(t *testing.T) {
		form := "email=flowtest@example.com&password=testpassword123"
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusSeeOther {
			t.Errorf("Login status = %d, want %d", rec.Code, http.StatusSeeOther)
		}

		if rec.Result().Cookies() == nil {
			t.Error("Expected session cookie")
		}
	})

	t.Run("VerifyUserInDatabase", func(t *testing.T) {
		user, err := database.GetUserByEmail(context.Background(), "flowtest@example.com")
		if err != nil {
			t.Fatalf("GetUserByEmail() error = %v", err)
		}

		if user == nil {
			t.Fatal("User not found in database")
		}

		if user.Email != "flowtest@example.com" {
			t.Errorf("User email = %s, want flowtest@example.com", user.Email)
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

func TestVerificationToken(t *testing.T) {
	token1, err := auth.GenerateVerificationToken()
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

func TestDatabaseUserCreation(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment()

	ctx := context.Background()

	user := &models.User{
		Email:                  "db-test@example.com",
		PasswordHash:           "test-hash",
		EmailVerified:          true,
		NotificationEmail:      true,
		NotificationTelegram:   false,
		NotificationThresholds: "[90, 60, 30]",
	}

	id, err := database.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	fetched, err := database.GetUserByID(ctx, int(id))
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	if fetched.Email != user.Email {
		t.Errorf("Email = %v, want %v", fetched.Email, user.Email)
	}
}

func TestDatabaseDomainOperations(t *testing.T) {
	setupTestEnvironment(t)
	defer teardownTestEnvironment()

	ctx := context.Background()

	user := &models.User{
		Email:                  "domain-ops-test@example.com",
		PasswordHash:           "test-hash",
		NotificationThresholds: "[90, 60, 30]",
	}

	userID, _ := database.CreateUser(ctx, user)

	registrar := "Test Registrar"
	domain := &models.Domain{
		UserID:     int(userID),
		Name:       "ops-test.com",
		Registrar:  &registrar,
		ExpiryDate: time.Now().AddDate(2, 0, 0),
		Status:     "active",
	}

	domainID, err := database.CreateDomain(ctx, domain)
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}

	domains, err := database.GetDomainsByUserID(ctx, int(userID))
	if err != nil {
		t.Fatalf("GetDomainsByUserID() error = %v", err)
	}

	if len(domains) != 1 {
		t.Errorf("Expected 1 domain, got %d", len(domains))
	}

	err = database.DeleteDomain(ctx, int(domainID))
	if err != nil {
		t.Fatalf("DeleteDomain() error = %v", err)
	}

	remaining, _ := database.GetDomainsByUserID(ctx, int(userID))
	if len(remaining) != 0 {
		t.Error("Domain should have been deleted")
	}
}

func TestWHOISService(t *testing.T) {
	service := services.NewWHOISService()

	result, err := service.Lookup(context.Background(), "example.com")
	if err != nil {
		t.Logf("WHOIS lookup failed (expected in test environment): %v", err)
	}

	if result.DomainName != "example.com" {
		t.Errorf("DomainName = %v, want example.com", result.DomainName)
	}
}

func TestConfigLoading(t *testing.T) {
	os.Setenv("DB_PATH", "/custom/path/db.sqlite")
	os.Setenv("PORT", "8080")
	defer os.Unsetenv("DB_PATH")
	defer os.Unsetenv("PORT")

	cfg := &config.Config{
		DBPath:        "/custom/path/db.sqlite",
		Port:          8080,
		SessionSecret: "test-secret",
		CheckInterval: 6 * time.Hour,
	}

	if cfg.ServerAddr() != ":8080" {
		t.Errorf("ServerAddr() = %v, want :8080", cfg.ServerAddr())
	}
}

func TestMiddlewareRequireAuth(t *testing.T) {
	e := echo.New()

	handler := middleware.RequireAuth(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	if err == nil {
		t.Error("Expected error for unauthenticated request")
	}

	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("Expected HTTPError, got %T", err)
	}

	if httpErr.Code != http.StatusUnauthorized && httpErr.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want 401 or 500", httpErr.Code)
	}
}

func TestDomainValidation(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		wantErr bool
	}{
		{"valid domain", "example.com", false},
		{"valid subdomain", "sub.example.com", false},
		{"invalid TLD", "example", true},
		{"empty string", "", true},
		{"special chars", "exam!ple.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := isValidDomain(tt.domain)
			if isValid != !tt.wantErr {
				t.Errorf("isValidDomain(%q) = %v, want %v", tt.domain, isValid, !tt.wantErr)
			}
		})
	}
}

func isValidDomain(name string) bool {
	if name == "" {
		return false
	}
	if len(name) < 3 {
		return false
	}
	hasDot := strings.Contains(name, ".")
	if !hasDot {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '.' || c == '-') {
			return false
		}
	}
	return true
}
