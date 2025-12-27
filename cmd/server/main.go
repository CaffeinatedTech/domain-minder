package main

import (
	"log"
	"net/http"
	"os"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/handlers"
	"github.com/CaffeinatedTech/domain-minder/internal/middleware"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
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

	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())

	middleware.SetupSessionMiddleware(e, cfg.SessionSecret)

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	authHandler := handlers.NewAuthHandler(cfg)
	settingsHandler := handlers.NewSettingsHandler()

	e.GET("/register", authHandler.ShowRegister)
	e.POST("/register", authHandler.Register)
	e.GET("/login", authHandler.ShowLogin)
	e.POST("/login", authHandler.Login)
	e.GET("/verify", authHandler.VerifyEmail)

	protected := e.Group("")
	protected.Use(middleware.RequireAuth)
	protected.POST("/logout", authHandler.Logout)
	protected.GET("/verify/resend", authHandler.ResendVerification)
	protected.GET("/settings", settingsHandler.ShowSettings)
	protected.POST("/settings/notifications", settingsHandler.UpdateNotifications)
	protected.POST("/settings/thresholds", settingsHandler.UpdateThresholds)

	protected.GET("/dashboard", func(c echo.Context) error {
		user := middleware.GetCurrentUser(c)
		return c.String(http.StatusOK, `
        <html><body>
        <h1>Dashboard</h1>
        <p>Welcome, `+user.Email+`!</p>
        `+buildEmailVerificationBanner(user)+`
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

func buildEmailVerificationBanner(user *models.User) string {
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
