package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/J0es1ick/shortli/internal/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type UrlRepository struct {
	db *sqlx.DB
}

func NewUrlRepository(db *sqlx.DB) *UrlRepository {
	return &UrlRepository{
		db: db,
	}
}

func (r *UrlRepository) SaveUrl(url *models.URL) (int64, error) {
    query := `
        INSERT INTO url_info 
            (original_url, short_code, user_id, click_count, created_at) 
        VALUES ($1, $2, $3, $4, $5)
        RETURNING url_id
    `
    
    var id int64
    err := r.db.QueryRow(
        query,
        url.OriginalURL,
        url.ShortCode,
        url.UserID,
        url.ClickCount,
        url.CreatedAt,
    ).Scan(&id)
    
    if err != nil {
        var pgErr *pq.Error
        if errors.As(err, &pgErr) && pgErr.Code == "23505" {
            return 0, fmt.Errorf("url with this code already exists")
        }
        return 0, fmt.Errorf("insert value error: %v", err)
    }
    
    return id, nil
}

func (r *UrlRepository) FindAllUrl(limit, offset int) ([]models.URL, error) {
    query := `
        SELECT 
            url_id, 
            original_url, 
            short_code, 
            user_id,
            click_count, 
            created_at 
        FROM url_info
        LIMIT $1 OFFSET $2
    `

    urls := []models.URL{}
    err := r.db.Select(&urls, query, limit, offset)

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("url not found")
        }
        return nil, fmt.Errorf("select error: %v", err)
    }

    if len(urls) == 0 {
        return nil, fmt.Errorf("no URLs found")
    }
    
    return urls, nil
}

func (r *UrlRepository) FindUrlsByUserID(userID int, limit, offset int) ([]models.URL, error) {
    query := `
        SELECT 
            url_id, 
            original_url, 
            short_code, 
            user_id,
            click_count, 
            created_at 
        FROM url_info 
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

    urls := []models.URL{}
    err := r.db.Select(&urls, query, userID, limit, offset)

    if err != nil {
        return nil, fmt.Errorf("select user urls error: %v", err)
    }

    return urls, nil
}

func (r *UrlRepository) GetTotalUrlsByUserID(userID int) (int, error) {
    var count int
    err := r.db.QueryRow(`
        SELECT COUNT(*) 
        FROM url_info 
        WHERE user_id = $1
    `, userID).Scan(&count)
    
    if err != nil {
        return 0, fmt.Errorf("count user urls error: %v", err)
    }

    return count, nil
}

func (r *UrlRepository) GetTotalClicks() (int, error) {
    var total int
    err := r.db.QueryRow("SELECT COALESCE(SUM(click_count), 0) FROM url_info").Scan(&total)
    if err != nil {
        return 0, fmt.Errorf("get total clicks error: %v", err)
    }
    return total, nil
}

func (r *UrlRepository) GetTotalUrls() (int, error) {
    var count int
    err := r.db.QueryRow("SELECT COUNT(*) FROM url_info").Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("count error: %w", err)
    }

    return count, nil
}

func (r *UrlRepository) FindUrlByCode(code string) (*models.URL, error) {
    query := `
        SELECT 
            url_id, 
            original_url, 
            short_code, 
            user_id,
            click_count, 
            created_at 
        FROM url_info 
        WHERE short_code = $1
    `
    
    url := &models.URL{}
    err := r.db.QueryRow(query, code).Scan(
        &url.ID,
        &url.OriginalURL,
        &url.ShortCode,
        &url.UserID,
        &url.ClickCount,
        &url.CreatedAt,
    )
    
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("url not found")
        }
        return nil, fmt.Errorf("select error: %v", err)
    }
    
    return url, nil
}

func (r *UrlRepository) FindUrlByOriginalUrl(originalUrl string) (*models.URL, error) {
    query := `
        SELECT 
            url_id, 
            original_url, 
            short_code, 
            user_id,
            click_count, 
            created_at 
        FROM url_info 
        WHERE original_url = $1
    `

    url := &models.URL{}
    err := r.db.QueryRow(query, originalUrl).Scan(
        &url.ID,
        &url.OriginalURL,
        &url.ShortCode,
        &url.UserID,
        &url.ClickCount,
        &url.CreatedAt,
    )

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("url not found")
        }
        return nil, fmt.Errorf("select error: %v", err)
    }
    
    return url, nil
}

func (r *UrlRepository) UpdateUrlByCode(url *models.URL) error {
    query := `
        UPDATE url_info 
        SET 
            original_url = $1, 
            click_count = $2,
            created_at = $3
        WHERE short_code = $4
    `
    
    result, err := r.db.Exec(
        query,
        url.OriginalURL,
        url.ClickCount,
        url.CreatedAt,
        url.ShortCode,
    )
    
    if err != nil {
        return fmt.Errorf("update value error: %v", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %v", err)
    }
    
    if rowsAffected == 0 {
        return fmt.Errorf("no rows updated - url with code '%s' not found", url.ShortCode)
    }
    
    return nil
}

func (r *UrlRepository) DeleteUrlByCode(code string) error {
    query := `
        DELETE FROM url_info 
        WHERE short_code = $1
        RETURNING url_id
    `
    
    var deletedID int64
    err := r.db.QueryRow(query, code).Scan(&deletedID)
    
    if err != nil {
        if err == sql.ErrNoRows {
            return fmt.Errorf("url with code '%s' not found", code)
        }
        return fmt.Errorf("delete value error: %v", err)
    }
    
    return nil
}

func (r *UrlRepository)DeleteOldUrls() (int64, error) {
    query := `
        DELETE FROM url_info 
        WHERE created_at < NOW() - INTERVAL '1 month'
        RETURNING url_id
    `

    rows, err := r.db.Query(query)
    if err != nil {
        return 0, fmt.Errorf("delete old urls error: %v", err)
    }
    defer rows.Close()
    
    var count int64
    for rows.Next() {
        var id int64
        if err := rows.Scan(&id); err != nil {
            return 0, fmt.Errorf("scan deleted id error: %v", err)
        }
        count++
    }
    
    if err := rows.Err(); err != nil {
        return 0, fmt.Errorf("rows error: %v", err)
    }
    
    return count, nil
}

func (r *UrlRepository) SearchUrls(query string, limit, offset int) ([]models.URL, error) {
    searchQuery := `
        SELECT 
            url_id, 
            original_url, 
            short_code, 
            user_id,
            click_count, 
            created_at 
        FROM url_info
        WHERE original_url ILIKE $1 OR short_code ILIKE $2
        ORDER BY created_at DESC
        LIMIT $3 OFFSET $4
    `

    urls := []models.URL{}
    searchPattern := "%" + query + "%"
    err := r.db.Select(&urls, searchQuery, searchPattern, searchPattern, limit, offset)

    if err != nil {
        return nil, fmt.Errorf("search error: %v", err)
    }

    return urls, nil
}

func (r *UrlRepository) GetTotalSearchUrls(query string) (int, error) {
    var count int
    searchPattern := "%" + query + "%"
    err := r.db.QueryRow(`
        SELECT COUNT(*) 
        FROM url_info 
        WHERE original_url ILIKE $1 OR short_code ILIKE $2
    `, searchPattern, searchPattern).Scan(&count)
    
    if err != nil {
        return 0, fmt.Errorf("search count error: %v", err)
    }

    return count, nil
}