package models

import "time"

type URL struct {
	ID          int        `db:"url_id" json:"url_id,omitempty"`
	OriginalURL string     `db:"original_url" json:"original_url,omitempty"`
	ShortCode   string     `db:"short_code" json:"short_code,omitempty"`
	UserID      *int       `db:"user_id" json:"user_id,omitempty"`
	ClickCount  int        `db:"click_count" json:"click_count"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at,omitempty"`
	ExpiresAt   *time.Time `db:"expires_at" json:"expires_at"`
	IsActive    bool       `db:"is_active" json:"is_active"`
}

type ClickEvent struct {
	EventKey     string
	URLID        int
	ClickedAt    time.Time
	DeviceType   string
	Browser      string
	OS           string
	ReferrerHost string
	CountryCode  string
	IPHash       string
}

type AnalyticsBucket struct {
	Label string `db:"label" json:"label"`
	Count int    `db:"count" json:"count"`
}

type AnalyticsSummary struct {
	TotalClicks      int               `json:"total_clicks"`
	UniqueClicks     int               `json:"unique_clicks"`
	Daily            []AnalyticsBucket `json:"daily"`
	Devices          []AnalyticsBucket `json:"devices"`
	Browsers         []AnalyticsBucket `json:"browsers"`
	OperatingSystems []AnalyticsBucket `json:"operating_systems"`
	Referrers        []AnalyticsBucket `json:"referrers"`
	Countries        []AnalyticsBucket `json:"countries"`
}
