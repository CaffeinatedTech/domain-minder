package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/services/mailer"
	"github.com/CaffeinatedTech/domain-minder/internal/services/turnstile"
	"github.com/CaffeinatedTech/domain-minder/internal/templates"
	"github.com/labstack/echo/v4"
)

func setupTestHandler(t *testing.T) (*echo.Echo, *config.Config) {
	e := echo.New()
	e.Renderer = templates.NewRenderer("/home/adam/projects/domain-minder/internal/templates")

	cfg := &config.Config{
		DBPath:        "/tmp/domain_minder_test.db",
		Port:          9000,
		SessionSecret: "test-secret",
		CheckInterval: 6 * time.Hour,
		TurnstileConfig: config.TurnstileConfig{
			SiteKey:   "",
			SecretKey: "",
		},
	}

	return e, cfg
}

func TestShowRegister(t *testing.T) {
	e, cfg := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/register", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	m := mailer.NewService(cfg)
	ts := turnstile.NewService(&cfg.TurnstileConfig)
	h := NewAuthHandler(cfg, m, ts)
	if err := h.ShowRegister(c); err != nil {
		t.Fatalf("ShowRegister() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Register") && !strings.Contains(body, "register") {
		t.Error("Response should contain 'Register'")
	}
}

func TestShowLogin(t *testing.T) {
	e, cfg := setupTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	m := mailer.NewService(cfg)
	ts := turnstile.NewService(&cfg.TurnstileConfig)
	h := NewAuthHandler(cfg, m, ts)
	if err := h.ShowLogin(c); err != nil {
		t.Fatalf("ShowLogin() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Login") && !strings.Contains(body, "login") {
		t.Error("Response should contain 'Login'")
	}
}

func TestRegisterValidation(t *testing.T) {
	e, cfg := setupTestHandler(t)

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

			m := mailer.NewService(cfg)
			ts := turnstile.NewService(&cfg.TurnstileConfig)
			h := NewAuthHandler(cfg, m, ts)
			h.Register(c)

			if rec.Code != tt.wantStatus {
				t.Errorf("Status code = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestLoginValidation(t *testing.T) {
	e, cfg := setupTestHandler(t)

	tests := []struct {
		name       string
		email      string
		password   string
		wantStatus int
	}{
		{
			name:       "empty email",
			email:      "",
			password:   "password123",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty password",
			email:      "test@example.com",
			password:   "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := "email=" + tt.email + "&password=" + tt.password
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			m := mailer.NewService(cfg)
			ts := turnstile.NewService(&cfg.TurnstileConfig)
			h := NewAuthHandler(cfg, m, ts)
			h.Login(c)

			if rec.Code != tt.wantStatus {
				t.Errorf("Status code = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
