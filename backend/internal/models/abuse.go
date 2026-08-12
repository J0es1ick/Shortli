package models

import "time"

type AbuseReport struct {
	ID             int64      `db:"report_id" json:"report_id"`
	URLID          *int64     `db:"url_id" json:"url_id,omitempty"`
	ShortCode      string     `db:"short_code" json:"short_code"`
	OriginalURL    string     `db:"original_url" json:"original_url,omitempty"`
	ReporterEmail  *string    `db:"reporter_email" json:"reporter_email,omitempty"`
	ReporterIPHash string     `db:"reporter_ip_hash" json:"-"`
	Reason         string     `db:"reason" json:"reason"`
	Details        string     `db:"details" json:"details"`
	Status         string     `db:"status" json:"status"`
	ResolutionNote string     `db:"resolution_note" json:"resolution_note"`
	ReviewedBy     *int       `db:"reviewed_by" json:"reviewed_by,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	ReviewedAt     *time.Time `db:"reviewed_at" json:"reviewed_at,omitempty"`
}

type BlockedDomain struct {
	ID        int64     `db:"domain_id" json:"domain_id"`
	Domain    string    `db:"domain" json:"domain"`
	Reason    string    `db:"reason" json:"reason"`
	CreatedBy *int      `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
