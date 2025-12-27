package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/config"
	"github.com/CaffeinatedTech/domain-minder/internal/models"
)

func setupTestDB(t *testing.T) {
	os.Remove("/tmp/domain_minder_test.db")

	cfg := &config.Config{
		DBPath: "/tmp/domain_minder_test.db",
	}

	if err := Init(cfg); err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
}

func teardownTestDB() {
	Close()
	os.Remove("/tmp/domain_minder_test.db")
}

func TestDatabaseInit(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	if DB == nil {
		t.Error("Database should be initialized")
	}

	if err := DB.Ping(); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestUserCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	ctx := context.Background()

	user := &models.User{
		Email:                  "test@example.com",
		PasswordHash:           "hash",
		EmailVerified:          false,
		NotificationEmail:      true,
		NotificationTelegram:   false,
		NotificationThresholds: "[90, 60, 30]",
	}

	id, err := CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	user.ID = int(id)

	fetched, err := GetUserByID(ctx, int(id))
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	if fetched.Email != user.Email {
		t.Errorf("Email = %v, want %v", fetched.Email, user.Email)
	}

	byEmail, err := GetUserByEmail(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail() error = %v", err)
	}

	if byEmail.ID != user.ID {
		t.Errorf("User ID = %v, want %v", byEmail.ID, user.ID)
	}
}

func TestDomainCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	ctx := context.Background()

	user := &models.User{
		Email:                  "domain-test@example.com",
		PasswordHash:           "hash",
		NotificationThresholds: "[90, 60, 30]",
	}
	userID, _ := CreateUser(ctx, user)

	registrar := "Test Registrar"
	domain := &models.Domain{
		UserID:     int(userID),
		Name:       "example.com",
		Registrar:  &registrar,
		ExpiryDate: time.Now().AddDate(1, 0, 0),
		Status:     "active",
	}

	domainID, err := CreateDomain(ctx, domain)
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}

	domain.ID = int(domainID)

	fetched, err := GetDomainByID(ctx, int(domainID))
	if err != nil {
		t.Fatalf("GetDomainByID() error = %v", err)
	}

	if fetched.Name != domain.Name {
		t.Errorf("Name = %v, want %v", fetched.Name, domain.Name)
	}

	domains, err := GetDomainsByUserID(ctx, int(userID))
	if err != nil {
		t.Fatalf("GetDomainsByUserID() error = %v", err)
	}

	if len(domains) != 1 {
		t.Errorf("Number of domains = %d, want 1", len(domains))
	}
}

func TestGetUserByVerificationToken(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	ctx := context.Background()

	token := "test-verification-token"
	user := &models.User{
		Email:                  "token-test@example.com",
		PasswordHash:           "hash",
		EmailVerified:          false,
		EmailVerificationToken: &token,
	}

	id, err := CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	fetched, err := GetUserByVerificationToken(ctx, token)
	if err != nil {
		t.Fatalf("GetUserByVerificationToken() error = %v", err)
	}

	if fetched == nil {
		t.Fatal("User should be found by verification token")
	}

	if fetched.ID != int(id) {
		t.Errorf("User ID = %v, want %v", fetched.ID, id)
	}
}

func TestUpdateUserVerification(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	ctx := context.Background()

	user := &models.User{
		Email:         "verify-test@example.com",
		PasswordHash:  "hash",
		EmailVerified: false,
	}

	id, err := CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	err = UpdateUserVerification(ctx, int(id), true)
	if err != nil {
		t.Fatalf("UpdateUserVerification() error = %v", err)
	}

	fetched, err := GetUserByID(ctx, int(id))
	if err != nil {
		t.Fatalf("GetUserByID() error = %v", err)
	}

	if !fetched.EmailVerified {
		t.Error("User should be marked as verified")
	}
}

func TestGetAllActiveDomains(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	ctx := context.Background()

	user := &models.User{
		Email:        "active-domains-test@example.com",
		PasswordHash: "hash",
	}

	userID, _ := CreateUser(ctx, user)

	for i := 0; i < 3; i++ {
		domain := &models.Domain{
			UserID:     int(userID),
			Name:       fmt.Sprintf("domain%d.com", i),
			ExpiryDate: time.Now().AddDate(1, 0, 0),
			Status:     "active",
		}
		CreateDomain(ctx, domain)
	}

	domain := &models.Domain{
		UserID:     int(userID),
		Name:       "inactive-domain.com",
		ExpiryDate: time.Now().AddDate(1, 0, 0),
		Status:     "inactive",
	}
	CreateDomain(ctx, domain)

	domains, err := GetAllActiveDomains(ctx)
	if err != nil {
		t.Fatalf("GetAllActiveDomains() error = %v", err)
	}

	if len(domains) != 3 {
		t.Errorf("Number of active domains = %d, want 3", len(domains))
	}
}

func TestDeleteDomain(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB()

	ctx := context.Background()

	user := &models.User{
		Email:        "delete-domain-test@example.com",
		PasswordHash: "hash",
	}

	userID, _ := CreateUser(ctx, user)

	domain := &models.Domain{
		UserID:     int(userID),
		Name:       "delete-me.com",
		ExpiryDate: time.Now().AddDate(1, 0, 0),
		Status:     "active",
	}

	domainID, err := CreateDomain(ctx, domain)
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}

	err = DeleteDomain(ctx, int(domainID))
	if err != nil {
		t.Fatalf("DeleteDomain() error = %v", err)
	}

	fetched, err := GetDomainByID(ctx, int(domainID))
	if err != nil {
		t.Fatalf("GetDomainByID() error = %v", err)
	}

	if fetched != nil {
		t.Error("Domain should have been deleted")
	}
}
