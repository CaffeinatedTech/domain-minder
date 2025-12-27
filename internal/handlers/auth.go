package handlers

import (
	"context"
	"net/http"
	"net/url"

	"github.com/CaffeinatedTech/domain-minder/internal/auth"
	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/middleware"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
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

	existing, err := database.GetUserByEmail(c.Request().Context(), email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "database error")
	}
	if existing != nil {
		return c.String(http.StatusBadRequest, "Email already registered")
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
	}

	token, err := auth.GenerateVerificationToken()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

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

	verificationURL := h.buildVerificationURL(c, token)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), h.config.CheckInterval)
		defer cancel()
		if err := auth.SendVerificationEmail(ctx, h.config, email, verificationURL); err != nil {
			println("Failed to send verification email:", err.Error())
		}
	}()

	sess, _ := session.Get("session", c)
	sess.Values["user_id"] = user.ID
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (h *AuthHandler) ShowRegister(c echo.Context) error {
	return c.Render(http.StatusOK, "register", nil)
}

func (h *AuthHandler) ShowLogin(c echo.Context) error {
	return c.Render(http.StatusOK, "login", nil)
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

	sess, _ := session.Get("session", c)
	sess.Values["user_id"] = user.ID
	sess.Save(c.Request(), c.Response())

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

	return c.Render(http.StatusOK, "verified", nil)
}

func (h *AuthHandler) ResendVerification(c echo.Context) error {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	if user.EmailVerified {
		return c.String(http.StatusBadRequest, "Email already verified")
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
