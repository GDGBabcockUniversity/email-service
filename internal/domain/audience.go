package domain

import (
	"time"
)

type Audience struct {
	ID        string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Email     string    `gorm:"type:text" json:"email"`
	CreatedAt time.Time `gorm:"type:timestamptz" json:"created_at"`
	ProjectID string    `gorm:"type:uuid" json:"project_id"`
}
