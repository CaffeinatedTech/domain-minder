package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type banEntry struct {
	attempts int
	banUntil time.Time
	lastSeen time.Time
}

type BruteForceConfig struct {
	MaxAttempts     int
	BaseBanDuration time.Duration
	MaxBanDuration  time.Duration
	CleanupInterval time.Duration
}

var (
	banStore = make(map[string]*banEntry)
	banMu    sync.RWMutex
	config   = BruteForceConfig{
		MaxAttempts:     5,
		BaseBanDuration: 15 * time.Minute,
		MaxBanDuration:  24 * time.Hour,
		CleanupInterval: 5 * time.Minute,
	}
)

func SetupBruteForceProtection() {
	go cleanupLoop()
}

func cleanupLoop() {
	ticker := time.NewTicker(config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		banMu.Lock()
		now := time.Now()
		for ip, entry := range banStore {
			if now.After(entry.banUntil) && now.Sub(entry.lastSeen) > config.CleanupInterval {
				delete(banStore, ip)
			}
		}
		banMu.Unlock()
	}
}

func BruteForceCheck(ip string) (banned bool, remaining time.Duration) {
	banMu.RLock()
	defer banMu.RUnlock()

	entry, exists := banStore[ip]
	if !exists {
		return false, 0
	}

	if time.Now().Before(entry.banUntil) {
		return true, time.Until(entry.banUntil)
	}

	return false, 0
}

func RecordFailedAttempt(ip string) {
	banMu.Lock()
	defer banMu.Unlock()

	entry, exists := banStore[ip]
	if !exists {
		banStore[ip] = &banEntry{
			attempts: 1,
			banUntil: time.Time{},
			lastSeen: time.Now(),
		}
		return
	}

	entry.attempts++
	entry.lastSeen = time.Now()

	if entry.attempts >= config.MaxAttempts {
		duration := config.BaseBanDuration
		for i := 1; i < entry.attempts; i++ {
			duration *= 2
			if duration >= config.MaxBanDuration {
				duration = config.MaxBanDuration
				break
			}
		}
		entry.banUntil = time.Now().Add(duration)
	}
}

func RecordSuccessfulLogin(ip string) {
	banMu.Lock()
	defer banMu.Unlock()

	delete(banStore, ip)
}

func BruteForceMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			if banned, remaining := BruteForceCheck(ip); banned {
				isHTMX := c.Request().Header.Get("HX-Request") == "true"
				errorMsg := "Too many failed attempts. Try again in " + formatDuration(remaining)

				if isHTMX {
					return c.HTML(http.StatusForbidden, `<div class="alert alert-error" style="margin-bottom: 1rem;">`+errorMsg+`</div>`)
				}

				return c.Render(http.StatusForbidden, "login", map[string]interface{}{
					"Error": errorMsg,
				})
			}

			return next(c)
		}
	}
}

func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	if minutes < 60 {
		return formatInt(minutes) + "m"
	}
	hours := int(d.Hours())
	if hours < 24 {
		return formatInt(hours) + "h"
	}
	return formatInt(int(d.Hours()/24)) + "d"
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
