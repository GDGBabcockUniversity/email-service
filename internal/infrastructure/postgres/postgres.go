package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/config"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const OwnerID = "00000000-0000-4000-8000-000000000001"

func Connect(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(pg.Open(cfg.DatabaseURL()), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := db.AutoMigrate(&domain.User{}, &domain.Project{}, &domain.Template{}, &domain.ApiKey{}, &domain.Email{}, &domain.Audience{}); err != nil {
		sqlDB.Close()
		return nil, err
	}

	owner := domain.User{ID: OwnerID, Username: "internal-admin", Email: "admin@localhost.invalid", Password: "!", Role: domain.ROLE_ADMIN}

	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&owner).Error; err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func repositoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return domain.ErrNotFound
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return domain.ErrConflict
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	default:
		return fmt.Errorf("%w: %v", domain.ErrUnavailable, err)
	}
}

func changed(rows int64, err error) error {
	if err != nil {
		return repositoryError(err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
