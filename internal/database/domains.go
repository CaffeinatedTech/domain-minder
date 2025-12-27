package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/CaffeinatedTech/domain-minder/internal/models"
)

func CreateDomain(ctx context.Context, domain *models.Domain) (int64, error) {
	result, err := DB.ExecContext(ctx, `
        INSERT INTO domains (user_id, name, registrar, expiry_date, whois_raw, status, notes)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, domain.UserID, domain.Name, domain.Registrar, domain.ExpiryDate, domain.WHOISRaw,
		domain.Status, domain.Notes)
	if err != nil {
		return 0, fmt.Errorf("failed to create domain: %w", err)
	}
	return result.LastInsertId()
}

func GetDomainByID(ctx context.Context, id int) (*models.Domain, error) {
	domain := &models.Domain{}
	err := DB.QueryRowContext(ctx, `
        SELECT id, user_id, name, registrar, expiry_date, whois_raw, last_checked,
            status, notes, created_at, updated_at
        FROM domains WHERE id = ?
    `, id).Scan(&domain.ID, &domain.UserID, &domain.Name, &domain.Registrar,
		&domain.ExpiryDate, &domain.WHOISRaw, &domain.LastChecked, &domain.Status,
		&domain.Notes, &domain.CreatedAt, &domain.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}
	return domain, nil
}

func GetDomainsByUserID(ctx context.Context, userID int) ([]*models.Domain, error) {
	rows, err := DB.QueryContext(ctx, `
        SELECT id, user_id, name, registrar, expiry_date, whois_raw, last_checked,
            status, notes, created_at, updated_at
        FROM domains WHERE user_id = ? ORDER BY expiry_date ASC
    `, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get domains: %w", err)
	}
	defer rows.Close()

	var domains []*models.Domain
	for rows.Next() {
		domain := &models.Domain{}
		if err := rows.Scan(&domain.ID, &domain.UserID, &domain.Name, &domain.Registrar,
			&domain.ExpiryDate, &domain.WHOISRaw, &domain.LastChecked, &domain.Status,
			&domain.Notes, &domain.CreatedAt, &domain.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, rows.Err()
}

func GetAllActiveDomains(ctx context.Context) ([]*models.Domain, error) {
	rows, err := DB.QueryContext(ctx, `
        SELECT id, user_id, name, registrar, expiry_date, whois_raw, last_checked,
            status, notes, created_at, updated_at
        FROM domains WHERE status = 'active'
    `)
	if err != nil {
		return nil, fmt.Errorf("failed to get all domains: %w", err)
	}
	defer rows.Close()

	var domains []*models.Domain
	for rows.Next() {
		domain := &models.Domain{}
		if err := rows.Scan(&domain.ID, &domain.UserID, &domain.Name, &domain.Registrar,
			&domain.ExpiryDate, &domain.WHOISRaw, &domain.LastChecked, &domain.Status,
			&domain.Notes, &domain.CreatedAt, &domain.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, rows.Err()
}

func UpdateDomain(ctx context.Context, domain *models.Domain) error {
	_, err := DB.ExecContext(ctx, `
        UPDATE domains SET name = ?, registrar = ?, expiry_date = ?, whois_raw = ?,
            last_checked = ?, status = ?, notes = ?, updated_at = ?
        WHERE id = ?
    `, domain.Name, domain.Registrar, domain.ExpiryDate, domain.WHOISRaw,
		domain.LastChecked, domain.Status, domain.Notes, time.Now(), domain.ID)
	return err
}

func UpdateDomainWHOIS(ctx context.Context, id int, expiryDate time.Time, registrar, whoisRaw string) error {
	_, err := DB.ExecContext(ctx, `
        UPDATE domains SET expiry_date = ?, registrar = ?, whois_raw = ?,
            last_checked = ?, updated_at = ?
        WHERE id = ?
    `, expiryDate, registrar, whoisRaw, time.Now(), time.Now(), id)
	return err
}

func DeleteDomain(ctx context.Context, id int) error {
	_, err := DB.ExecContext(ctx, "DELETE FROM domains WHERE id = ?", id)
	return err
}
