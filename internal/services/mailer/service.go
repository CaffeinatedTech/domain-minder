package mailer

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/database"
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

type Service struct {
	config *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{config: cfg}
}

func (s *Service) EnqueueVerificationEmail(ctx context.Context, toEmail, verificationURL string) error {
	body, err := s.renderTemplate("verification", map[string]string{
		"VerificationURL": verificationURL,
	})
	if err != nil {
		return err
	}
	return database.EnqueueEmail(ctx, toEmail, "Verify your Domain Minder email", body)
}

func (s *Service) EnqueueNotificationEmail(ctx context.Context, toEmail, domainName, registrar, expiryDate string, daysRemaining int) error {
	body, err := s.renderTemplate("notification", map[string]string{
		"DomainName":    domainName,
		"Registrar":     registrar,
		"ExpiryDate":    expiryDate,
		"DaysRemaining": fmt.Sprintf("%d", daysRemaining),
		"DashboardURL":  fmt.Sprintf("http://localhost:%d/dashboard", s.config.Port),
	})
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("Domain %s expires in %d days", domainName, daysRemaining)
	return database.EnqueueEmail(ctx, toEmail, subject, body)
}

func (s *Service) renderTemplate(name string, data interface{}) (string, error) {
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
