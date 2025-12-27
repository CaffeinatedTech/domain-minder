package database

import (
	"context"
	"database/sql"
	"time"
)

type EmailJob struct {
	ID          int
	ToEmail     string
	Subject     string
	Body        string
	Status      string
	Attempts    int
	LastAttempt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func EnqueueEmail(ctx context.Context, toEmail, subject, body string) error {
	query := `INSERT INTO email_queue (to_email, subject, body, status, attempts) VALUES (?, ?, ?, 'pending', 0)`
	_, err := DB.ExecContext(ctx, query, toEmail, subject, body)
	return err
}

func GetPendingEmails(ctx context.Context, limit int, maxRetries int) ([]*EmailJob, error) {
	query := `
        SELECT id, to_email, subject, body, status, attempts, last_attempt, created_at, updated_at
        FROM email_queue
        WHERE status = 'pending' OR (status = 'failed' AND attempts < ?)
        ORDER BY created_at ASC
        LIMIT ?
    `
	rows, err := DB.QueryContext(ctx, query, maxRetries, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*EmailJob
	for rows.Next() {
		var job EmailJob
		var lastAttempt sql.NullTime
		if err := rows.Scan(
			&job.ID, &job.ToEmail, &job.Subject, &job.Body,
			&job.Status, &job.Attempts, &lastAttempt,
			&job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if lastAttempt.Valid {
			job.LastAttempt = &lastAttempt.Time
		}
		jobs = append(jobs, &job)
	}
	return jobs, nil
}

func UpdateEmailStatus(ctx context.Context, id int, status string, errorMessage string) error {
	query := `
        UPDATE email_queue
        SET status = ?, error_message = ?, last_attempt = CURRENT_TIMESTAMP, attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
	_, err := DB.ExecContext(ctx, query, status, errorMessage, id)
	return err
}
