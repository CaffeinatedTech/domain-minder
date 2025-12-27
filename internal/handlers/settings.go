package handlers

import (
	"net/http"

	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/middleware"
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
    <p>Email: `+user.Email+` (`+emailStatus+`)</p>
    <h2>Notification Preferences</h2>
    <form method="POST" action="/settings/notifications">
        <label>
            <input type="checkbox" name="notification_email" `+boolChecked(user.NotificationEmail)+`>
            Email notifications
        </label><br>
        <label>
            <input type="checkbox" name="notification_telegram" `+boolChecked(user.NotificationTelegram)+`>
            Telegram notifications
        </label><br>
        <label>
            Telegram Chat ID: <input type="text" name="telegram_chat_id" value="`+nullString(user.TelegramChatID)+`">
        </label><br>
        <button type="submit">Save</button>
    </form>
    <h2>Notification Thresholds</h2>
    <form method="POST" action="/settings/thresholds">
        <label>
            Days before expiry (comma-separated):
            <input type="text" name="thresholds" value="`+user.NotificationThresholds+`">
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
