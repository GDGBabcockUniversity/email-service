package domain

import (
	"time"
)

type Audience struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email     string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"type:timestamptz"`
	ProjectID string    `gorm:"type:uuid"`
}
