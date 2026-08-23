package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/J0es1ick/shortli/internal/models"
	"github.com/jmoiron/sqlx"
)

const urlColumns = `
    url_id,
    original_url,
    short_code,
    user_id,
    click_count,
    created_at,
    expires_at,
    is_active
`

type UrlRepository struct {
	db *sqlx.DB
}

func NewUrlRepository(db *sqlx.DB) *UrlRepository {
	return &UrlRepository{
		db: db,
	}
}

func (r *UrlRepository) Health(ctx context.Context) error {
	if err := r.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check: %w", err)
	}
	return nil
}

func (r *UrlRepository) SaveUrl(ctx context.Context, url *models.URL) (int64, error) {
	query := `
		INSERT INTO url_info
			(original_url, short_code, user_id, click_count, created_at, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING url_id
    `

	var id int64
	err := r.db.QueryRowContext(ctx,
		query,
		url.OriginalURL,
		url.ShortCode,
		url.UserID,
		url.ClickCount,
		url.CreatedAt,
		url.ExpiresAt,
		url.IsActive,
	).Scan(&id)

	if err != nil {
		if isUniqueViolation(err) {
			return 0, fmt.Errorf("url with this code already exists")
		}
		return 0, fmt.Errorf("insert value error: %v", err)
	}

	return id, nil
}

func (r *UrlRepository) FindUrlByOriginalForOwner(ctx context.Context, originalURL string, userID *int) (*models.URL, error) {
	url := &models.URL{}
	err := r.db.GetContext(ctx, url, `
		SELECT `+urlColumns+`
		FROM url_info
		WHERE original_url = $1
		  AND user_id IS NOT DISTINCT FROM $2::INTEGER
		  AND ($2::INTEGER IS NOT NULL OR (is_active = TRUE AND (expires_at IS NULL OR expires_at > NOW())))
		ORDER BY created_at DESC, url_id DESC
		LIMIT 1
	`, originalURL, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("url not found: %w", sql.ErrNoRows)
		}
		return nil, fmt.Errorf("find URL by owner and destination: %w", err)
	}
	return url, nil
}

func (r *UrlRepository) FindOrSaveUrl(ctx context.Context, url *models.URL) (*models.URL, bool, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, false, fmt.Errorf("begin idempotent URL creation: %w", err)
	}
	defer tx.Rollback()

	ownerKey := "guest"
	if url.UserID != nil {
		ownerKey = strconv.Itoa(*url.UserID)
	}
	lockKey := fmt.Sprintf("%d:%s%s", len(ownerKey), ownerKey, url.OriginalURL)
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return nil, false, fmt.Errorf("lock URL creation: %w", err)
	}

	existing := &models.URL{}
	err = tx.GetContext(ctx, existing, `
		SELECT `+urlColumns+`
		FROM url_info
		WHERE original_url = $1
		  AND user_id IS NOT DISTINCT FROM $2::INTEGER
		  AND ($2::INTEGER IS NOT NULL OR (is_active = TRUE AND (expires_at IS NULL OR expires_at > NOW())))
		ORDER BY created_at DESC, url_id DESC
		LIMIT 1
	`, url.OriginalURL, url.UserID)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, false, fmt.Errorf("commit existing URL lookup: %w", err)
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("find existing URL during creation: %w", err)
	}

	err = tx.QueryRowContext(ctx, `
		INSERT INTO url_info
			(original_url, short_code, user_id, click_count, created_at, expires_at, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING url_id
	`,
		url.OriginalURL,
		url.ShortCode,
		url.UserID,
		url.ClickCount,
		url.CreatedAt,
		url.ExpiresAt,
		url.IsActive,
	).Scan(&url.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, false, fmt.Errorf("url with this code already exists")
		}
		return nil, false, fmt.Errorf("insert URL during idempotent creation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit URL creation: %w", err)
	}
	return url, true, nil
}

func (r *UrlRepository) FindAllUrl(ctx context.Context, limit, offset int) ([]models.URL, error) {
	query := `SELECT ` + urlColumns + ` FROM url_info ORDER BY created_at DESC LIMIT $1 OFFSET $2`

	urls := []models.URL{}
	err := r.db.SelectContext(ctx, &urls, query, limit, offset)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("url not found: %w", err)
		}
		return nil, fmt.Errorf("select error: %v", err)
	}

	return urls, nil
}

func (r *UrlRepository) FindUrlsByUserID(ctx context.Context, userID int, limit, offset int) ([]models.URL, error) {
	query := `SELECT ` + urlColumns + `
		FROM url_info
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	urls := []models.URL{}
	err := r.db.SelectContext(ctx, &urls, query, userID, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("select user urls error: %v", err)
	}

	return urls, nil
}

func (r *UrlRepository) GetTotalUrlsByUserID(ctx context.Context, userID int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) 
        FROM url_info 
        WHERE user_id = $1
    `, userID).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("count user urls error: %v", err)
	}

	return count, nil
}

func (r *UrlRepository) GetTotalClicks(ctx context.Context) (int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(click_count), 0) FROM url_info").Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get total clicks error: %v", err)
	}
	return total, nil
}

func (r *UrlRepository) GetTotalUrls(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM url_info").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count error: %w", err)
	}

	return count, nil
}

func (r *UrlRepository) FindUrlByCode(ctx context.Context, code string) (*models.URL, error) {
	url := &models.URL{}
	err := r.db.GetContext(ctx, url, `SELECT `+urlColumns+` FROM url_info WHERE short_code = $1`, code)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("url not found: %w", err)
		}
		return nil, fmt.Errorf("select error: %v", err)
	}

	return url, nil
}

func (r *UrlRepository) FindUrlByOriginalUrl(ctx context.Context, originalUrl string) (*models.URL, error) {
	url := &models.URL{}
	err := r.db.GetContext(ctx, url, `SELECT `+urlColumns+` FROM url_info WHERE original_url = $1 ORDER BY created_at DESC LIMIT 1`, originalUrl)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("url not found")
		}
		return nil, fmt.Errorf("select error: %v", err)
	}

	return url, nil
}

func (r *UrlRepository) UpdateUrlSettings(ctx context.Context, code string, isActive bool, expiresAt *time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE url_info
		SET is_active = $1, expires_at = $2
		WHERE short_code = $3
	`, isActive, expiresAt, code)

	if err != nil {
		return fmt.Errorf("update value error: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated - url with code '%s' not found", code)
	}

	return nil
}

func (r *UrlRepository) DeleteUrlByCode(ctx context.Context, code string) error {
	query := `
        DELETE FROM url_info 
        WHERE short_code = $1
        RETURNING url_id
    `

	var deletedID int64
	err := r.db.QueryRowContext(ctx, query, code).Scan(&deletedID)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("url with code '%s' not found", code)
		}
		return fmt.Errorf("delete value error: %v", err)
	}

	return nil
}

func (r *UrlRepository) DeactivateExpiredUrls(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		UPDATE url_info
		SET is_active = FALSE
		WHERE is_active = TRUE AND expires_at IS NOT NULL AND expires_at <= NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("deactivate expired urls: %w", err)
	}
	return result.RowsAffected()
}

func (r *UrlRepository) SearchUrls(ctx context.Context, query string, limit, offset int) ([]models.URL, error) {
	searchQuery := `SELECT ` + urlColumns + `
		FROM url_info
        WHERE original_url ILIKE $1 OR short_code ILIKE $2
        ORDER BY created_at DESC
        LIMIT $3 OFFSET $4
    `

	urls := []models.URL{}
	searchPattern := "%" + query + "%"
	err := r.db.SelectContext(ctx, &urls, searchQuery, searchPattern, searchPattern, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("search error: %v", err)
	}

	return urls, nil
}

func (r *UrlRepository) RecordClick(event *models.ClickEvent) error {
	return r.RecordClickContext(context.Background(), event)
}

func (r *UrlRepository) RecordClickContext(ctx context.Context, event *models.ClickEvent) error {
	_, err := r.db.ExecContext(ctx, `
		WITH inserted AS (
			INSERT INTO click_event
				(event_key, url_id, clicked_at, device_type, browser, os, referrer_host, country_code, ip_hash)
			SELECT $1, url_id, $3, $4, $5, $6, $7, $8, $9
			FROM url_info
			WHERE url_id = $2
			ON CONFLICT (event_key) WHERE event_key IS NOT NULL DO NOTHING
			RETURNING url_id
		)
		UPDATE url_info
		SET click_count = click_count + 1
		WHERE url_id IN (SELECT url_id FROM inserted)
	`, event.EventKey, event.URLID, event.ClickedAt, event.DeviceType, event.Browser, event.OS, event.ReferrerHost, event.CountryCode, event.IPHash)
	if err != nil {
		return fmt.Errorf("record click: %w", err)
	}
	return nil
}

func (r *UrlRepository) GetAnalytics(ctx context.Context, urlID int, since time.Time) (models.AnalyticsSummary, error) {
	result := models.AnalyticsSummary{
		Daily: []models.AnalyticsBucket{}, Devices: []models.AnalyticsBucket{},
		Browsers: []models.AnalyticsBucket{}, OperatingSystems: []models.AnalyticsBucket{},
		Referrers: []models.AnalyticsBucket{}, Countries: []models.AnalyticsBucket{},
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT NULLIF(ip_hash, ''))
		FROM click_event WHERE url_id = $1 AND clicked_at >= $2
	`, urlID, since).Scan(&result.TotalClicks, &result.UniqueClicks); err != nil {
		return result, fmt.Errorf("analytics totals: %w", err)
	}

	if err := r.db.SelectContext(ctx, &result.Daily, `
		SELECT TO_CHAR(clicked_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS label, COUNT(*) AS count
		FROM click_event WHERE url_id = $1 AND clicked_at >= $2
		GROUP BY label ORDER BY label
	`, urlID, since); err != nil {
		return result, fmt.Errorf("daily analytics: %w", err)
	}

	var err error
	if result.Devices, err = r.analyticsBreakdown(ctx, urlID, since, "device_type"); err != nil {
		return result, err
	}
	if result.Browsers, err = r.analyticsBreakdown(ctx, urlID, since, "browser"); err != nil {
		return result, err
	}
	if result.OperatingSystems, err = r.analyticsBreakdown(ctx, urlID, since, "os"); err != nil {
		return result, err
	}
	if result.Referrers, err = r.analyticsBreakdown(ctx, urlID, since, "referrer_host"); err != nil {
		return result, err
	}
	if result.Countries, err = r.analyticsBreakdown(ctx, urlID, since, "country_code"); err != nil {
		return result, err
	}
	return result, nil
}

func (r *UrlRepository) analyticsBreakdown(ctx context.Context, urlID int, since time.Time, column string) ([]models.AnalyticsBucket, error) {
	allowed := map[string]bool{"device_type": true, "browser": true, "os": true, "referrer_host": true, "country_code": true}
	if !allowed[column] {
		return nil, fmt.Errorf("unsupported analytics dimension")
	}
	items := []models.AnalyticsBucket{}
	query := fmt.Sprintf(`
		SELECT COALESCE(NULLIF(%s, ''), 'Unknown') AS label, COUNT(*) AS count
		FROM click_event WHERE url_id = $1 AND clicked_at >= $2
		GROUP BY label ORDER BY count DESC, label LIMIT 8
	`, column)
	if err := r.db.SelectContext(ctx, &items, query, urlID, since); err != nil {
		return nil, fmt.Errorf("analytics breakdown %s: %w", column, err)
	}
	return items, nil
}

func (r *UrlRepository) GetTotalSearchUrls(ctx context.Context, query string) (int, error) {
	var count int
	searchPattern := "%" + query + "%"
	err := r.db.QueryRowContext(ctx, `
        SELECT COUNT(*) 
        FROM url_info 
        WHERE original_url ILIKE $1 OR short_code ILIKE $2
    `, searchPattern, searchPattern).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("search count error: %v", err)
	}

	return count, nil
}
