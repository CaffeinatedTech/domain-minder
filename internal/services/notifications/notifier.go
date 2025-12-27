package notifications

import (
	"context"
)

type Notification struct {
	UserID        int
	DomainID      int
	DomainName    string
	Registrar     string
	ExpiryDate    string
	DaysRemaining int
	Channel       string
}

type Notifier interface {
	Send(ctx context.Context, notification *Notification) error
	Name() string
	CanSend() bool
}
