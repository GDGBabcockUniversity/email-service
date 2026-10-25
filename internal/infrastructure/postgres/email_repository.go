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

// FindByProjectID returns a project's most recent emails, newest first, optionally
// narrowed to a single status. A non-positive limit falls back to 50.
func (r *EmailRepository) FindByProjectID(ctx context.Context, projectID string, status domain.Status, limit int) ([]domain.Email, error) {
	if limit <= 0 {
		limit = 50
	}
	query := gorm.G[domain.Email](r.db).Where("project_id = ?", projectID)
	if status != "" {
		query = query.Where("status = ?", string(status))
	}
	emails, err := query.Order("created_at DESC, id DESC").Limit(limit).Find(ctx)
	return emails, repositoryError(err)
}

// CountByStatus tallies a project's emails per delivery status.
func (r *EmailRepository) CountByStatus(ctx context.Context, projectID string) (map[domain.Status]int64, error) {
	var rows []struct {
		Status string
		Total  int64
	}
	err := r.db.WithContext(ctx).Model(&domain.Email{}).
		Select("status, count(*) AS total").
		Where("project_id = ?", projectID).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, repositoryError(err)
	}
	counts := make(map[domain.Status]int64, len(rows))
	for _, row := range rows {
		counts[domain.Status(row.Status)] = row.Total
	}
	return counts, nil
}
