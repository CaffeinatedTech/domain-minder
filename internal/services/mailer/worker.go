package mailer

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/database"
)

const (
	BatchSize    = 10
	MaxRetries   = 3
	PollInterval = 10 * time.Second
)

func (s *Service) StartWorker() {
	log.Println("Email queue worker started")
	ticker := time.NewTicker(PollInterval)
	go func() {
		for range ticker.C {
			if err := s.processQueue(); err != nil {
				log.Printf("Error processing email queue: %v", err)
			}
		}
	}()
}

func (s *Service) processQueue() error {
	ctx := context.Background()
	jobs, err := database.GetPendingEmails(ctx, BatchSize, MaxRetries)
	if err != nil {
		return fmt.Errorf("failed to get pending emails: %w", err)
	}

	for _, job := range jobs {
		err := s.sendEmail(job.ToEmail, job.Subject, job.Body)

		status := "sent"
		errMsg := ""
		if err != nil {
			status = "failed"
			errMsg = err.Error()
			log.Printf("Failed to send email to %s: %v", job.ToEmail, err)
		} else {
			log.Printf("Successfully sent email to %s", job.ToEmail)
		}

		if err := database.UpdateEmailStatus(ctx, job.ID, status, errMsg); err != nil {
			log.Printf("Failed to update status for job %d: %v", job.ID, err)
		}
	}
	return nil
}

func (s *Service) sendEmail(toEmail, subject, body string) error {
	if s.config.SMTPConfig.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	auth := smtp.PlainAuth("", s.config.SMTPConfig.Username, s.config.SMTPConfig.Password, s.config.SMTPConfig.Host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		s.config.SMTPConfig.From, toEmail, subject, body)

	addr := fmt.Sprintf("%s:%d", s.config.SMTPConfig.Host, s.config.SMTPConfig.Port)
	return smtp.SendMail(addr, auth, s.config.SMTPConfig.From, []string{toEmail}, []byte(msg))
}
