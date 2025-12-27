package services

import (
	"time"
)

type WHOISResult struct {
	DomainName string
	Registrar  string
	ExpiryDate time.Time
	WHOISRaw   string
	Error      error
}

type WHOISService struct{}

func NewWHOISService() *WHOISService {
	return &WHOISService{}
}

func (s *WHOISService) Lookup(domain string) (*WHOISResult, error) {
	return &WHOISResult{
		DomainName: domain,
		Error:      nil,
	}, nil
}
