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

	telegramChatID := ""
	if user.TelegramChatID != nil {
		telegramChatID = *user.TelegramChatID
	}

	return c.Render(http.StatusOK, "settings", map[string]interface{}{
		"User":           user,
		"TelegramChatID": telegramChatID,
	})
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
