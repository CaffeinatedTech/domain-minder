package auth

import (
	"testing"
)

func TestGenerateVerificationToken(t *testing.T) {
	token1, err := GenerateVerificationToken()
	if err != nil {
		t.Fatalf("GenerateVerificationToken() error = %v", err)
	}

	if len(token1) != 64 {
		t.Errorf("Token length = %d, want 64", len(token1))
	}

	token2, err := GenerateVerificationToken()
	if err != nil {
		t.Fatalf("GenerateVerificationToken() error = %v", err)
	}

	if token1 == token2 {
		t.Error("Tokens should be unique")
	}
}

func TestGenerateVerificationTokenFormat(t *testing.T) {
	token, _ := GenerateVerificationToken()

	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Token contains invalid character: %c", c)
		}
	}
}

func TestGenerateVerificationTokenUniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := GenerateVerificationToken()
		if err != nil {
			t.Fatalf("GenerateVerificationToken() error = %v", err)
		}
		if tokens[token] {
			t.Error("Duplicate token generated")
		}
		tokens[token] = true
	}
}
