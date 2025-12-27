package services

import (
	"testing"
	"time"
)

func TestExtractRegistrar(t *testing.T) {
	s := NewWHOISService()

	tests := []struct {
		name     string
		raw      string
		expected string
	}{
		{
			name:     "Registrar with colon",
			raw:      "Registrar: Example Registrar Inc.",
			expected: "Example Registrar Inc.",
		},
		{
			name:     "Registrar Name format",
			raw:      "Registrar Name: Another Registrar LLC",
			expected: "Another Registrar LLC",
		},
		{
			name:     "Registrar WHOIS Server",
			raw:      "Registrar WHOIS Server: whois.example.com",
			expected: "whois.example.com",
		},
		{
			name:     "Whois Server variation",
			raw:      "Whois Server: ns1.example.com",
			expected: "ns1.example.com",
		},
		{
			name:     "Mixed case",
			raw:      "REGISTRAR: Test Registrar",
			expected: "Test Registrar",
		},
		{
			name:     "No registrar found",
			raw:      "Domain Name: example.com",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.extractRegistrar(tt.raw)
			if result != tt.expected {
				t.Errorf("extractRegistrar() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractExpiryDate(t *testing.T) {
	s := NewWHOISService()

	tests := []struct {
		name     string
		raw      string
		expected time.Time
	}{
		{
			name:     "Expiry Date YYYY-MM-DD",
			raw:      "Expiry Date: 2025-12-31",
			expected: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Expiration Date YYYY-MM-DD",
			raw:      "Expiration Date: 2026-06-15",
			expected: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Registry Expiry Date",
			raw:      "Registry Expiry Date: 2027-03-20",
			expected: time.Date(2027, 3, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Valid Until format",
			raw:      "Valid Until: 2028-01-01",
			expected: time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "DD-Mon-YYYY format",
			raw:      "Expiry date: 31-Dec-2025",
			expected: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Expiration DD-Mon-YYYY format",
			raw:      "Expiration Date: 15-Jun-2026",
			expected: time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "DD/MM/YYYY format",
			raw:      "Expiry Date: 31/12/2025",
			expected: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "YYYY.MM.DD format",
			raw:      "Expiry Date: 2025.12.31",
			expected: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Lowercase expires",
			raw:      "expires: 2025-12-31",
			expected: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "No date found",
			raw:      "Domain Name: example.com",
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.extractExpiryDate(tt.raw)
			if !result.Equal(tt.expected) {
				t.Errorf("extractExpiryDate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

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

func TestGetDefaultServer(t *testing.T) {
	s := NewWHOISService()

	tests := []struct {
		domain   string
		expected string
	}{
		{"example.com", "whois.verisign-grs.com"},
		{"example.net", "whois.verisign-grs.com"},
		{"example.org", "whois.pir.org"},
		{"example.io", "whois.nic.io"},
		{"example.uk", "whois.nic.uk"},
		{"example.de", "whois.denic.de"},
		{"unknown.xyz", "whois.iana.org"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := s.getDefaultServer(tt.domain)
			if result != tt.expected {
				t.Errorf("getDefaultServer(%q) = %q, want %q", tt.domain, result, tt.expected)
			}
		})
	}
}

func TestExtractFromRealWhoisResponse(t *testing.T) {
	s := NewWHOISService()

	rawWhois := `Domain Name: EXAMPLE.COM
Registrar: Example Registrar, Inc.
Registrar WHOIS Server: whois.example.com
Domain Status: clientTransferProhibited https://icann.org/epp#clientTransferProhibited
Name Server: NS1.EXAMPLE.COM
Name Server: NS2.EXAMPLE.COM
DNSSEC: unsigned
Expiration Date: 2026-12-31
`

	result := &WHOISResult{
		DomainName: "example.com",
		WHOISRaw:   rawWhois,
	}

	result.Registrar = s.extractRegistrar(rawWhois)
	result.ExpiryDate = s.extractExpiryDate(rawWhois)

	if result.Registrar != "Example Registrar, Inc." {
		t.Errorf("Registrar = %q, want %q", result.Registrar, "Example Registrar, Inc.")
	}

	expectedExpiry := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	if !result.ExpiryDate.Equal(expectedExpiry) {
		t.Errorf("ExpiryDate = %v, want %v", result.ExpiryDate, expectedExpiry)
	}
}
