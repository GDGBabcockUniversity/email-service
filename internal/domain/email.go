package domain

import "time"

type Email struct {
	ID            string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	To            string     `gorm:"not null" json:"to"`
	ProjectID     string     `gorm:"type:uuid;not null" json:"project_id"`
	TemplateID    string     `gorm:"type:uuid;not null" json:"template_id"`
	Subject       string     `gorm:"not null" json:"subject"`
	Body          string     `gorm:"type:text;not null" json:"-"`
	Status        Status     `gorm:"not null" json:"status"`
	Attempts      int        `gorm:"not null;default:0" json:"attempts"`
	LastError     string     `json:"last_error,omitempty"`
	NextAttemptAt *time.Time `json:"-"`
	CreatedAt     time.Time  `json:"created_at"`
	SentAt        *time.Time `json:"sent_at"`
}

type Status string

const (
	StatusPending    Status = "PENDING"
	StatusProcessing Status = "PROCESSING"
	StatusSent       Status = "SENT"
	StatusFailed     Status = "FAILED"
)
