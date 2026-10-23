package domain

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"uniqueIndex:project_owner_name;not null" json:"name"`
	Description string         `json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	UserID      string         `gorm:"type:uuid;uniqueIndex:project_owner_name;not null" json:"user_id"`
	Templates   []Template     `gorm:"foreignKey:ProjectID" json:"-"`
	ApiKeys     []ApiKey       `gorm:"foreignKey:ProjectID" json:"-"`
	Audiences   []Audience     `gorm:"foreignKey:ProjectID" json:"-"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
