package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type MaintenanceRepository struct {
	db *sqlx.DB
}

type MaintenanceResult struct {
	ExpiredLinks    int64
	ExpiredSessions int64
	OldClickEvents  int64
	OldReports      int64
}

func NewMaintenanceRepository(db *sqlx.DB) *MaintenanceRepository {
	return &MaintenanceRepository{db: db}
}

func (r *MaintenanceRepository) Run(
	ctx context.Context,
	analyticsRetentionDays int,
	reportRetentionDays int,
) (MaintenanceResult, error) {
	result := MaintenanceResult{}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("begin maintenance: %w", err)
	}
	defer tx.Rollback()

	statement := func(destination *int64, query string, args ...interface{}) error {
		execResult, execErr := tx.ExecContext(ctx, query, args...)
		if execErr != nil {
			return execErr
		}
		*destination, execErr = execResult.RowsAffected()
		return execErr
	}
	if err := statement(&result.ExpiredLinks, `
		UPDATE url_info SET is_active = FALSE
		WHERE is_active = TRUE AND expires_at IS NOT NULL AND expires_at <= NOW()
	`); err != nil {
		return result, fmt.Errorf("deactivate expired links: %w", err)
	}
	if err := statement(&result.ExpiredSessions, `
		DELETE FROM session_info WHERE expires_at <= NOW()
	`); err != nil {
		return result, fmt.Errorf("delete expired sessions: %w", err)
	}
	if err := statement(&result.OldClickEvents, `
		DELETE FROM click_event
		WHERE clicked_at < NOW() - ($1 * INTERVAL '1 day')
	`, analyticsRetentionDays); err != nil {
		return result, fmt.Errorf("delete old click events: %w", err)
	}
	if err := statement(&result.OldReports, `
		DELETE FROM abuse_report
		WHERE status <> 'pending'
		  AND reviewed_at < NOW() - ($1 * INTERVAL '1 day')
	`, reportRetentionDays); err != nil {
		return result, fmt.Errorf("delete old abuse reports: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit maintenance: %w", err)
	}
	return result, nil
}
