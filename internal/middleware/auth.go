package middleware

import (
	"net/http"

	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

const SessionUserKey = "user_id"

func SetupSessionMiddleware(e *echo.Echo, secret string) {
	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 2,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	e.Use(session.Middleware(store))
}

func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("session", c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "session error")
		}

		userID := sess.Values[SessionUserKey]
		if userID == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
		}

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
