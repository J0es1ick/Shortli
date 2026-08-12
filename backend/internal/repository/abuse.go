package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
	"github.com/jmoiron/sqlx"
)

type AbuseRepository struct {
	db *sqlx.DB
}

func NewAbuseRepository(db *sqlx.DB) *AbuseRepository {
	return &AbuseRepository{db: db}
}

func (r *AbuseRepository) Create(report *models.AbuseReport) error {
	err := r.db.QueryRow(`
		INSERT INTO abuse_report
			(url_id, short_code, reporter_email, reporter_ip_hash, reason, details)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING report_id, created_at, status
	`, report.URLID, report.ShortCode, report.ReporterEmail, report.ReporterIPHash, report.Reason, report.Details).
		Scan(&report.ID, &report.CreatedAt, &report.Status)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("a pending report for this link already exists")
		}
		return fmt.Errorf("create abuse report: %w", err)
	}
	return nil
}

func (r *AbuseRepository) FindByID(id int64) (*models.AbuseReport, error) {
	report := &models.AbuseReport{}
	err := r.db.Get(report, `
		SELECT ar.report_id, ar.url_id, ar.short_code,
			COALESCE(u.original_url, '') AS original_url,
			ar.reporter_email, ar.reporter_ip_hash, ar.reason, ar.details,
			ar.status, ar.resolution_note, ar.reviewed_by, ar.created_at, ar.reviewed_at
		FROM abuse_report ar
		LEFT JOIN url_info u ON u.url_id = ar.url_id
		WHERE ar.report_id = $1
	`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("abuse report not found")
		}
		return nil, fmt.Errorf("find abuse report: %w", err)
	}
	return report, nil
}

func (r *AbuseRepository) List(status string, limit, offset int) ([]models.AbuseReport, int, error) {
	reports := []models.AbuseReport{}
	args := []interface{}{limit, offset}
	where := ""
	if status != "" && status != "all" {
		where = "WHERE ar.status = $3"
		args = append(args, status)
	}
	query := `
		SELECT ar.report_id, ar.url_id, ar.short_code,
			COALESCE(u.original_url, '') AS original_url,
			ar.reporter_email, ar.reporter_ip_hash, ar.reason, ar.details,
			ar.status, ar.resolution_note, ar.reviewed_by, ar.created_at, ar.reviewed_at
		FROM abuse_report ar
		LEFT JOIN url_info u ON u.url_id = ar.url_id
		` + where + `
		ORDER BY CASE WHEN ar.status = 'pending' THEN 0 ELSE 1 END, ar.created_at DESC
		LIMIT $1 OFFSET $2
	`
	if err := r.db.Select(&reports, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list abuse reports: %w", err)
	}

	countQuery := `SELECT COUNT(*) FROM abuse_report`
	countArgs := []interface{}{}
	if status != "" && status != "all" {
		countQuery += ` WHERE status = $1`
		countArgs = append(countArgs, status)
	}
	var total int
	if err := r.db.Get(&total, countQuery, countArgs...); err != nil {
		return nil, 0, fmt.Errorf("count abuse reports: %w", err)
	}
	return reports, total, nil
}

func (r *AbuseRepository) Resolve(id int64, status, note string, reviewedBy int) error {
	result, err := r.db.Exec(`
		UPDATE abuse_report
		SET status = $1, resolution_note = $2, reviewed_by = $3, reviewed_at = $4
		WHERE report_id = $5
	`, status, note, reviewedBy, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("resolve abuse report: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("abuse report not found")
	}
	return nil
}

func (r *AbuseRepository) IsDomainBlocked(domain string) (bool, error) {
	domain = normalizeDomain(domain)
	var blocked bool
	err := r.db.Get(&blocked, `
		SELECT EXISTS (
			SELECT 1 FROM blocked_domain
			WHERE $1 = domain OR $1 LIKE '%.' || domain
		)
	`, domain)
	if err != nil {
		return false, fmt.Errorf("check blocked domain: %w", err)
	}
	return blocked, nil
}

func (r *AbuseRepository) BlockDomain(domain, reason string, createdBy int) error {
	domain = normalizeDomain(domain)
	if domain == "" {
		return fmt.Errorf("domain is required")
	}
	_, err := r.db.Exec(`
		INSERT INTO blocked_domain(domain, reason, created_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (domain) DO UPDATE
		SET reason = EXCLUDED.reason, created_by = EXCLUDED.created_by, created_at = NOW()
	`, domain, reason, createdBy)
	if err != nil {
		return fmt.Errorf("block domain: %w", err)
	}
	return nil
}

func (r *AbuseRepository) ListBlockedDomains() ([]models.BlockedDomain, error) {
	items := []models.BlockedDomain{}
	if err := r.db.Select(&items, `
		SELECT domain_id, domain, reason, created_by, created_at
		FROM blocked_domain ORDER BY created_at DESC
	`); err != nil {
		return nil, fmt.Errorf("list blocked domains: %w", err)
	}
	return items, nil
}

func (r *AbuseRepository) UnblockDomain(id int64) error {
	result, err := r.db.Exec(`DELETE FROM blocked_domain WHERE domain_id = $1`, id)
	if err != nil {
		return fmt.Errorf("unblock domain: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("blocked domain not found")
	}
	return nil
}

func normalizeDomain(value string) string {
	return strings.TrimPrefix(strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), "."), "www.")
}
