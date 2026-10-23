package postgres

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"gorm.io/gorm"
)

type EmailRepository struct{ db *gorm.DB }

func NewEmailRepository(db *gorm.DB) *EmailRepository { return &EmailRepository{db: db} }

func (r *EmailRepository) Save(ctx context.Context, email *domain.Email) error {
	return repositoryError(gorm.G[domain.Email](r.db).Create(ctx, email))
}

func (r *EmailRepository) FindByID(ctx context.Context, id string) (*domain.Email, error) {
	email, err := gorm.G[domain.Email](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, repositoryError(err)
	}
	return &email, nil
}

func (r *EmailRepository) Update(ctx context.Context, email *domain.Email) error {
	result := r.db.WithContext(ctx).Model(email).Select("Status", "Attempts", "LastError", "NextAttemptAt", "SentAt").Updates(email)
	return changed(result.RowsAffected, result.Error)
}

func (r *EmailRepository) SaveAll(ctx context.Context, emails []domain.Email) error {
	return repositoryError(gorm.G[domain.Email](r.db).CreateInBatches(ctx, &emails, 10))
}