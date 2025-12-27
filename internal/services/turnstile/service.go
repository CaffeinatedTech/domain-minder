package turnstile

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
)

const siteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type TurnstileResponse struct {
	Success     bool        `json:"success"`
	ChallengeTS string      `json:"challenge_ts"`
	Hostname    string      `json:"hostname"`
	ErrorCodes  []string    `json:"error-codes"`
	Action      string      `json:"action"`
	CData       string      `json:"cdata"`
	EphemeralID interface{} `json:"metadata,omitempty"`
}

type Service struct {
	secretKey string
	siteKey   string
	client    *http.Client
}

func NewService(cfg *config.TurnstileConfig) *Service {
	return &Service{
		secretKey: cfg.SecretKey,
		siteKey:   cfg.SiteKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) SiteKey() string {
	return s.siteKey
}

func (s *Service) ValidateToken(token, remoteip string) (*TurnstileResponse, error) {
	if token == "" {
		return &TurnstileResponse{
			Success:    false,
			ErrorCodes: []string{"missing-input-response"},
		}, nil
	}

	if s.secretKey == "" {
		return nil, fmt.Errorf("turnstile secret key not configured")
	}

	data := url.Values{}
	data.Set("secret", s.secretKey)
	data.Set("response", token)
	if remoteip != "" && remoteip != "127.0.0.1" && remoteip != "::1" {
		data.Set("remoteip", remoteip)
	}

	resp, err := s.client.PostForm(siteverifyURL, data)
	if err != nil {
		return nil, fmt.Errorf("turnstile validation request failed: %w", err)
	}
	defer resp.Body.Close()

	var result TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode turnstile response: %w", err)
	}

	return &result, nil
}

func (s *Service) IsEnabled() bool {
	return s.siteKey != "" && s.secretKey != ""
}
