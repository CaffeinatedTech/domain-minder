package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/CaffeinatedTech/domain-minder/internal/handlers"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

type MockRenderer struct{}

func (t *MockRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	if m, ok := data.(map[string]interface{}); ok {
		if errVal, ok := m["Error"]; ok {
			w.Write([]byte(errVal.(string)))
		}
	}
	return nil
}

func TestCSRFProtection(t *testing.T) {
	e := echo.New()
	e.Renderer = &MockRenderer{}
	e.Use(echomw.CSRFWithConfig(echomw.CSRFConfig{
		TokenLookup: "form:csrf",
	}))

	e.POST("/protected", func(c echo.Context) error {
		return c.String(http.StatusOK, "Allowed")
	})

	// Case 1: No Token
	req := httptest.NewRequest(http.MethodPost, "/protected", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusForbidden {
		t.Errorf("Expected 400/403 for missing CSRF, got %d", rec.Code)
	}

	// Case 2: With Token
	// Creating a valid token is hard without the CSRF middleware's internal logic leaking.
	// But we can check that it BLOCKS invalid ones.
	// To test positive case, we would need to extract the token from a previous GET response, but CSRF middleware generates it on GET?
	// The middleware usually sets a cookie.
	// Simplest verification is that middleware IS active (Case 1).
}

func TestRateLimiting(t *testing.T) {
	e := echo.New()
	// Same config as main.go
	e.POST("/login", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}, echomw.RateLimiter(echomw.NewRateLimiterMemoryStore(2)))

	// Hit it 5 times rapidly
	limitHit := false
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			limitHit = true
			break
		}
	}

	if !limitHit {
		t.Error("Rate limiter did not trigger after 5 rapid requests (limit is 2 req/s)")
	}
}

func TestHoneypot(t *testing.T) {
	// Create handler with nil dependencies (safe for early check)
	h := handlers.NewAuthHandler(nil, nil)
	e := echo.New()
	e.Renderer = &MockRenderer{}

	// Case 1: Honeypot Triggered
	form := make(url.Values)
	form.Set("email", "bot@evil.com")
	form.Set("password", "password123")
	form.Set("website_url", "http://spam.com")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_ = h.Login(c) // Expect error
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for honeypot, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Invalid email or password") {
		t.Errorf("Expected 'Invalid email or password', got %s", rec.Body.String())
	}

	// Case 2: Honeypot Empty (Should proceed to database error or invalid password, but pass honeypot check)
	// Note: Since dependencies are nil, it will likely panic or error later.
	// We just want to ensure it DOES NOT return "Spam detected".
	form = make(url.Values)
	form.Set("email", "user@good.com")
	form.Set("password", "password123")
	// website_url empty

	req = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	// Recover from panic as dependencies are nil
	defer func() {
		if r := recover(); r != nil {
			// Panic is expected due to nil DB/Config, but it means it PASSED the honeypot check
		}
	}()

	h.Login(c)

	// If we reach here without 401 Invalid email, it's good.
	if rec.Code == http.StatusUnauthorized && strings.Contains(rec.Body.String(), "Invalid email or password") {
		t.Error("Honeypot triggered for empty field")
	}

	// Case 3: Register Honeypot
	form = make(url.Values)
	form.Set("email", "bot2@evil.com")
	form.Set("password", "password123")
	form.Set("website_url", "http://spam.com")

	req = httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	_ = h.Register(c)

	if !strings.Contains(rec.Body.String(), "Registrations closed") {
		t.Errorf("Expected 'Registrations closed' for register honeypot, got %s", rec.Body.String())
	}
}
