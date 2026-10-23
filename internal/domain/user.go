package domain

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username  string         `gorm:"unique;not null" json:"username"`
	Password  string         `gorm:"not null" json:"-"`
	Email     string         `gorm:"unique;not null" json:"email"`
	JoinedAt  time.Time      `gorm:"not null;default:now()" json:"joined_at"`
	Role      Role           `gorm:"not null;default:USER" json:"role"`
	Projects  []Project      `gorm:"foreignKey:UserID" json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Role string

const (
	ROLE_USER        Role = "USER"
	ROLE_ADMIN       Role = "ADMIN"
	ROLE_SUPER_ADMIN Role = "SUPER_ADMIN"
)
