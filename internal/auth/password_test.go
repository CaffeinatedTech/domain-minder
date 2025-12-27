package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == password {
		t.Error("Hash should not equal password")
	}

	if len(hash) < 10 {
		t.Error("Hash too short")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword() should return true for correct password")
	}

	if CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword() should return false for wrong password")
	}
}

func TestHashPasswordUnique(t *testing.T) {
	password := "testpassword123"

	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	if hash1 == hash2 {
		t.Error("Same password should produce different hashes")
	}

	if !CheckPassword(password, hash1) {
		t.Error("First hash should validate")
	}
	if !CheckPassword(password, hash2) {
		t.Error("Second hash should validate")
	}
}

func TestCheckPasswordInvalidHash(t *testing.T) {
	password := "testpassword123"
	invalidHash := "not-a-valid-hash"

	if CheckPassword(password, invalidHash) {
		t.Error("CheckPassword() should return false for invalid hash")
	}
}
