package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
)

type TelegramNotifier struct {
	cfg    *config.Config
	client *http.Client
}

type TelegramMessage struct {
	ChatID      string            `json:"chat_id"`
	Text        string            `json:"text"`
	ParseMode   string            `json:"parse_mode,omitempty"`
	ReplyMarkup *TelegramKeyboard `json:"reply_markup,omitempty"`
}

type TelegramKeyboard struct {
	InlineKeyboard [][]TelegramButton `json:"inline_keyboard"`
}

type TelegramButton struct {
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

func NewTelegramNotifier(cfg *config.Config) *TelegramNotifier {
	return &TelegramNotifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (n *TelegramNotifier) Name() string {
	return "telegram"
}

func (n *TelegramNotifier) CanSend() bool {
	return n.cfg.TelegramConfig.BotToken != ""
}

func (n *TelegramNotifier) Send(ctx context.Context, chatID string, notification *Notification) error {
	if !n.CanSend() {
		return fmt.Errorf("Telegram bot token not configured")
	}

	if chatID == "" {
		return fmt.Errorf("Telegram chat ID not set")
	}

	text := fmt.Sprintf(`*Domain Expiry Warning*

Domain: *%s*
Registrar: %s
Expires in: *%d days* (%s)

Don't forget to renew!`,
		notification.DomainName,
		notification.Registrar,
		notification.DaysRemaining,
		notification.ExpiryDate,
	)

	msg := TelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
		ReplyMarkup: &TelegramKeyboard{
			InlineKeyboard: [][]TelegramButton{
				{
					{Text: "View Dashboard", URL: n.buildDashboardURL()},
				},
			},
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.cfg.TelegramConfig.BotToken)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

func (n *TelegramNotifier) buildDashboardURL() string {
	return fmt.Sprintf("http://localhost:%d/dashboard", n.cfg.Port)
}
