package handlers

import (
	"net/http"
	"net/url"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/auth"
	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/middleware"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
	"github.com/CaffeinatedTech/domain-minder/internal/services/mailer"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	config *config.Config
	mailer *mailer.Service
}

func NewAuthHandler(cfg *config.Config, m *mailer.Service) *AuthHandler {
	return &AuthHandler{config: cfg, mailer: m}
}

func (h *AuthHandler) Register(c echo.Context) error {
	isHTMX := c.Request().Header.Get("HX-Request") == "true"

	// Honeypot check
	if c.FormValue("website_url") != "" {
		if isHTMX {
			return c.HTML(http.StatusBadRequest, `<div class="alert alert-error" style="margin-bottom: 1rem;">Registrations closed</div>`)
		}
		return c.Render(http.StatusBadRequest, "register", map[string]interface{}{"Error": "Registrations closed"})
	}
	email := c.FormValue("email")
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")

	if email == "" || password == "" {
		return c.Render(http.StatusBadRequest, "register", map[string]interface{}{"Error": "Email and password are required"})
	}

	if password != confirmPassword {
		return c.Render(http.StatusBadRequest, "register", map[string]interface{}{"Error": "Passwords do not match"})
	}

	if len(password) < 8 {
		return c.Render(http.StatusBadRequest, "register", map[string]interface{}{"Error": "Password must be at least 8 characters"})
	}

	existing, err := database.GetUserByEmail(c.Request().Context(), email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "database error")
	}
	if existing != nil {
		return c.Render(http.StatusBadRequest, "register", map[string]interface{}{"Error": "Email already registered"})
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
	}

	token, err := auth.GenerateVerificationToken()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	user := &models.User{
		Email:                    email,
		PasswordHash:             hash,
		EmailVerified:            false,
		EmailVerificationToken:   &token,
		EmailVerificationExpires: &expiresAt,
		NotificationEmail:        true,
		NotificationTelegram:     false,
		NotificationThresholds:   "[90, 60, 30, 14, 7, 3, 1]",
	}

	id, err := database.CreateUser(c.Request().Context(), user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create user")
	}
	user.ID = int(id)

	verificationURL := h.buildVerificationURL(c, token)

	if err := h.mailer.EnqueueVerificationEmail(c.Request().Context(), email, verificationURL); err != nil {
		println("Failed to enqueue verification email:", err.Error())
	}

	sess, _ := session.Get("session", c)
	sess.Values = map[interface{}]interface{}{"user_id": user.ID}
	sess.Save(c.Request(), c.Response())

	if isHTMX {
		c.Response().Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (h *AuthHandler) ShowRegister(c echo.Context) error {
	return c.Render(http.StatusOK, "register", nil)
}

func (h *AuthHandler) ShowLogin(c echo.Context) error {
	return c.Render(http.StatusOK, "login", nil)
}

func (h *AuthHandler) Login(c echo.Context) error {
	ip := c.RealIP()
	isHTMX := c.Request().Header.Get("HX-Request") == "true"

	// Honeypot check
	if c.FormValue("website_url") != "" {
		middleware.RecordFailedAttempt(ip)
		if isHTMX {
			return c.HTML(http.StatusUnauthorized, `<div class="alert alert-error" style="margin-bottom: 1rem;">Invalid email or password</div>`)
		}
		return c.Render(http.StatusUnauthorized, "login", map[string]interface{}{"Error": "Invalid email or password"})
	}
	email := c.FormValue("email")
	password := c.FormValue("password")

	if email == "" || password == "" {
		return c.Render(http.StatusBadRequest, "login", map[string]interface{}{"Error": "Email and password are required"})
	}

	user, err := database.GetUserByEmail(c.Request().Context(), email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "database error")
	}
	if user == nil {
		middleware.RecordFailedAttempt(ip)
		if isHTMX {
			return c.HTML(http.StatusUnauthorized, `<div class="alert alert-error" style="margin-bottom: 1rem;">Invalid email or password</div>`)
		}
		return c.Render(http.StatusUnauthorized, "login", map[string]interface{}{"Error": "Invalid email or password"})
	}

	if !auth.CheckPassword(password, user.PasswordHash) {
		middleware.RecordFailedAttempt(ip)
		if isHTMX {
			return c.HTML(http.StatusUnauthorized, `<div class="alert alert-error" style="margin-bottom: 1rem;">Invalid email or password</div>`)
		}
		return c.Render(http.StatusUnauthorized, "login", map[string]interface{}{"Error": "Invalid email or password"})
	}

	middleware.RecordSuccessfulLogin(ip)

	sess, _ := session.Get("session", c)
	sess.Values = map[interface{}]interface{}{"user_id": user.ID}
	sess.Save(c.Request(), c.Response())

	if isHTMX {
		c.Response().Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}

	return c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (h *AuthHandler) Logout(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Values["user_id"] = nil
	sess.Save(c.Request(), c.Response())
	return c.Redirect(http.StatusSeeOther, "/login")
}

func (h *AuthHandler) VerifyEmail(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		return c.Render(http.StatusBadRequest, "login", map[string]interface{}{"Error": "Verification token required"})
	}

	user, err := database.GetUserByVerificationToken(c.Request().Context(), token)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "database error")
	}
	if user == nil {
		return c.Render(http.StatusBadRequest, "login", map[string]interface{}{"Error": "Invalid or expired verification token"})
	}

	if user.EmailVerificationExpires != nil && time.Now().After(*user.EmailVerificationExpires) {
		return c.Render(http.StatusBadRequest, "login", map[string]interface{}{"Error": "Verification token has expired"})
	}

	if err := database.UpdateUserVerification(c.Request().Context(), user.ID, true); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to verify email")
	}

	return c.Render(http.StatusOK, "verified", nil)
}

func (h *AuthHandler) ResendVerification(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	if user.EmailVerified {
		return c.Render(http.StatusBadRequest, "dashboard", map[string]interface{}{"Error": "Email already verified"})
	}

	token, err := auth.GenerateVerificationToken()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	user.EmailVerificationToken = &token
	if err := database.UpdateUser(c.Request().Context(), user); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update user")
	}

	verificationURL := h.buildVerificationURL(c, token)
	if err := h.mailer.EnqueueVerificationEmail(c.Request().Context(), user.Email, verificationURL); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to enqueue verification email")
	}

	return c.HTML(http.StatusOK, `<span style="color: var(--success); font-weight: 500;">&#10003; Sent!</span>`)
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
