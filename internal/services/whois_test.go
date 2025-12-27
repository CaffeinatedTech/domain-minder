package services

import (
	"testing"
	"time"
)

func TestCacheOperations(t *testing.T) {
	s := NewWHOISService()

	result := &WHOISResult{
		DomainName: "test.com",
		Registrar:  "Test Registrar",
		ExpiryDate: time.Now().Add(24 * time.Hour),
	}

	s.setCache("test.com", result)

	cached := s.getFromCache("test.com")
	if cached == nil {
		t.Fatal("Expected cached result, got nil")
	}
	if cached.Registrar != "Test Registrar" {
		t.Errorf("Registrar = %q, want %q", cached.Registrar, "Test Registrar")
	}

	s.ClearCache()

	afterClear := s.getFromCache("test.com")
	if afterClear != nil {
		t.Error("Expected nil after cache clear")
	}
}

func TestExtractWhoisServer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Registrar WHOIS Server",
			input:    "Registrar WHOIS Server: whois.godaddy.com\n",
			expected: "whois.godaddy.com",
		},
		{
			name:     "Whois Server",
			input:    "Whois Server: ns1.example.com\n",
			expected: "ns1.example.com",
		},
		{
			name:     "IANA whois format",
			input:    "whois: whois.nic.photography\n",
			expected: "whois.nic.photography",
		},
		{
			name:     "No server found",
			input:    "Domain Name: example.com\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWhoisServer(tt.input)
			if result != tt.expected {
				t.Errorf("extractWhoisServer() = %q, want %q", result, tt.expected)
			}
		})
	}
}
