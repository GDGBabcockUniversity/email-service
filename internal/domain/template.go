package domain

import (
	"time"
)

type Template struct {
	Id        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"uniqueIndex:project_template_name;not null" json:"name"`
	Subject   string         `gorm:"not null" json:"subject"`
	Data      []string       `gorm:"serializer:json;not null" json:"data"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	UserID    string         `gorm:"type:uuid;not null" json:"user_id"`
	ProjectID string         `gorm:"type:uuid;uniqueIndex:project_template_name;not null" json:"project_id"`
	CreatedAt time.Time      `json:"created_at"`
}
