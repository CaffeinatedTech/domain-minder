package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/database"
	"github.com/CaffeinatedTech/domain-minder/internal/services/notifications"
)

type Checker struct {
	cfg             *config.Config
	cron            *cron.Cron
	notificationMgr *notifications.NotificationManager
	whoisService    *WHOISService
	running         bool
	runningMu       sync.Mutex
	stopChan        chan struct{}
}

func NewChecker(cfg *config.Config, notificationMgr *notifications.NotificationManager, whoisService *WHOISService) *Checker {
	return &Checker{
		cfg:             cfg,
		notificationMgr: notificationMgr,
		whoisService:    whoisService,
		stopChan:        make(chan struct{}),
	}
}

func (c *Checker) Start() error {
	c.runningMu.Lock()
	if c.running {
		c.runningMu.Unlock()
		return nil
	}
	c.running = true
	c.runningMu.Unlock()

	c.cron = cron.New(cron.WithSeconds())

	_, err := c.cron.AddFunc(c.cfg.CheckInterval.String(), c.CheckAllDomains)
	if err != nil {
		return err
	}

	c.cron.Start()

	log.Println("Background checker started")

	go func() {
		time.Sleep(10 * time.Second)
		c.CheckAllDomains()
	}()

	return nil
}

func (c *Checker) Stop() {
	c.runningMu.Lock()
	if !c.running {
		c.runningMu.Unlock()
		return
	}
	c.running = false
	c.runningMu.Unlock()

	if c.cron != nil {
		c.cron.Stop()
	}

	close(c.stopChan)
	log.Println("Background checker stopped")
}

func (c *Checker) CheckAllDomains() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	log.Println("Starting domain check...")

	domains, err := database.GetAllActiveDomains(ctx)
	if err != nil {
		log.Printf("Failed to get domains: %v", err)
		return
	}

	log.Printf("Checking %d domains", len(domains))

	checked := 0
	notificationsSent := 0
	errors := 0

	for _, domain := range domains {
		select {
		case <-ctx.Done():
			log.Println("Domain check cancelled due to timeout")
			return
		case <-c.stopChan:
			log.Println("Domain check stopped")
			return
		default:
		}

		user, err := database.GetUserByID(ctx, domain.UserID)
		if err != nil || user == nil {
			log.Printf("Failed to get user for domain %s: %v", domain.Name, err)
			errors++
			continue
		}

		result, err := c.whoisService.Lookup(ctx, domain.Name)
		if err != nil {
			log.Printf("WHOIS lookup failed for %s: %v", domain.Name, err)
			errors++
		} else {
			if !result.ExpiryDate.IsZero() && !result.ExpiryDate.Equal(domain.ExpiryDate) {
				if err := database.UpdateDomainWHOIS(ctx, domain.ID, result.ExpiryDate, result.Registrar, result.WHOISRaw); err != nil {
					log.Printf("Failed to update domain %s: %v", domain.Name, err)
					errors++
				} else {
					log.Printf("Updated %s expiry date to %s", domain.Name, result.ExpiryDate.Format("2006-01-02"))
				}
			}
			checked++
		}

		logs, err := c.notificationMgr.CheckDomain(ctx, user, domain)
		if err != nil {
			log.Printf("Failed to check notifications for %s: %v", domain.Name, err)
			errors++
		} else {
			notificationsSent += len(logs)
		}
	}

	log.Printf("Domain check complete: %d checked, %d notifications sent, %d errors", checked, notificationsSent, errors)
}

func (c *Checker) RunOnce() {
	go c.CheckAllDomains()
}
