package domain

import "time"

type ApiKey struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Key       string     `gorm:"uniqueIndex;not null" json:"-"`
	ProjectID string     `gorm:"type:uuid;not null" json:"project_id"`
	CreatedAt time.Time  `json:"created_at"`
	Revoked   bool       `gorm:"not null;default:false" json:"revoked"`
	RevokedAt *time.Time `json:"revoked_at"`
}
