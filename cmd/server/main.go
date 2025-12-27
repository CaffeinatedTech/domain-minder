package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/handlers"
	"github.com/CaffeinatedTech/domain-minder/internal/middleware"
	"github.com/CaffeinatedTech/domain-minder/internal/services"
	"github.com/CaffeinatedTech/domain-minder/internal/templates"
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

	e.Renderer = templates.NewRenderer("internal/templates")
	e.Static("/static", "static")

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	authHandler := handlers.NewAuthHandler(cfg)
	settingsHandler := handlers.NewSettingsHandler()

	whoisService := services.NewWHOISService()
	domainHandler := handlers.NewDomainHandler(whoisService)

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

	protected.GET("/domains", domainHandler.ListDomains)
	protected.GET("/domains/list", domainHandler.ListDomainsPartial)
	protected.GET("/domains/new", domainHandler.ShowAddDomain)
	protected.POST("/domains", domainHandler.AddDomain)
	protected.GET("/domains/:id", domainHandler.ShowEditDomain)
	protected.POST("/domains/:id", domainHandler.UpdateDomain)
	protected.DELETE("/domains/:id/delete", domainHandler.DeleteDomain)
	protected.POST("/domains/:id/check", domainHandler.CheckDomain)

	protected.GET("/dashboard", func(c echo.Context) error {
		user := middleware.GetCurrentUser(c)

		domains, err := database.GetDomainsByUserID(c.Request().Context(), user.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get domains")
		}

		totalDomains := len(domains)
		expiringSoon := 0
		expired := 0

		for _, d := range domains {
			daysLeft := int(time.Until(d.ExpiryDate).Hours() / 24)
			if daysLeft < 0 {
				expired++
			} else if daysLeft < 30 {
				expiringSoon++
			}
		}

		return c.Render(http.StatusOK, "dashboard", map[string]interface{}{
			"User":         user,
			"TotalDomains": totalDomains,
			"ExpiringSoon": expiringSoon,
			"Expired":      expired,
		})
	})

	log.Printf("Starting server on port %d", cfg.Port)
	if err := e.Start(cfg.ServerAddr()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
