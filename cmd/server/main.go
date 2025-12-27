package main

import (
	"log"
	"os"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database
	if err := database.Init(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	log.Printf("Starting server on port %d", cfg.Port)
	if err := e.Start(cfg.ServerAddr()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
