package auth

import (
	"bytes"
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"text/template"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
)

var emailTemplates = map[string]string{
	"verification": `
<html>
<body>
<h2>Verify Your Email Address</h2>
<p>Thank you for registering with Domain Minder.</p>
<p>Click the link below to verify your email address:</p>
<p><a href="{{ .VerificationURL }}">Verify Email</a></p>
<p>This link will expire in 24 hours.</p>
<p>If you did not register for Domain Minder, please ignore this email.</p>
</body>
</html>
`,
	"notification": `
<html>
<body>
<h2>Domain Expiry Warning: {{ .DomainName }}</h2>
<p>Your domain <strong>{{ .DomainName }}</strong> registered with <strong>{{ .Registrar }}</strong></p>
<p>will expire in <strong>{{ .DaysRemaining }} days</strong> on {{ .ExpiryDate }}.</p>
<p>Log in to your <a href="{{ .DashboardURL }}">Domain Minder dashboard</a> for more details.</p>
</body>
</html>
`,
}

func SendVerificationEmail(ctx context.Context, cfg *config.Config, toEmail, verificationURL string) error {
	if cfg.SMTPConfig.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	tmpl, ok := emailTemplates["verification"]
	if !ok {
		return fmt.Errorf("verification template not found")
	}

	var buf bytes.Buffer
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return err
	}
	if err := t.Execute(&buf, map[string]string{"VerificationURL": verificationURL}); err != nil {
		return err
	}

	auth := smtp.PlainAuth("", cfg.SMTPConfig.Username, cfg.SMTPConfig.Password, cfg.SMTPConfig.Host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Verify your Domain Minder email\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		cfg.SMTPConfig.From, toEmail, buf.String())

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", cfg.SMTPConfig.Host, cfg.SMTPConfig.Port),
		auth,
		cfg.SMTPConfig.From,
		[]string{toEmail},
		[]byte(msg),
	)
}

func SendNotificationEmail(ctx context.Context, cfg *config.Config, toEmail, domainName, registrar, expiryDate string, daysRemaining int) error {
	if cfg.SMTPConfig.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	tmpl, ok := emailTemplates["notification"]
	if !ok {
		return fmt.Errorf("notification template not found")
	}

	var buf bytes.Buffer
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return err
	}
	if err := t.Execute(&buf, map[string]string{
		"DomainName":    domainName,
		"Registrar":     registrar,
		"ExpiryDate":    expiryDate,
		"DaysRemaining": fmt.Sprintf("%d", daysRemaining),
		"DashboardURL":  fmt.Sprintf("http://localhost:%d/dashboard", cfg.Port),
	}); err != nil {
		return err
	}

	auth := smtp.PlainAuth("", cfg.SMTPConfig.Username, cfg.SMTPConfig.Password, cfg.SMTPConfig.Host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: Domain %s expires in %d days\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		cfg.SMTPConfig.From, toEmail, domainName, daysRemaining, buf.String())

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", cfg.SMTPConfig.Host, cfg.SMTPConfig.Port),
		auth,
		cfg.SMTPConfig.From,
		[]string{toEmail},
		[]byte(msg),
	)
}

func ParseEmailTemplate(name string, data interface{}) (string, error) {
	tmpl, ok := emailTemplates[name]
	if !ok {
		return "", fmt.Errorf("template not found: %s", name)
	}
	var buf bytes.Buffer
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return "", err
	}
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
