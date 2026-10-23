package postgres

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"gorm.io/gorm"
)

type AudienceRepository struct {
	db *gorm.DB
}

func NewAudienceRepository(db *gorm.DB) *AudienceRepository {
	return &AudienceRepository{db: db}
}

func (r *AudienceRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.Audience, error) {

	audience, err := gorm.G[domain.Audience](r.db).Where("project_id = ?", projectID).Find(ctx)

	if err != nil {
		return []domain.Audience{}, err
	}

	return audience, nil
}

func (r *AudienceRepository) FindByID(ctx context.Context, id string) (*domain.Audience, error) {

	audience, err := gorm.G[domain.Audience](r.db).Where("id = ?", id).First(ctx)

	if err != nil {
		return nil, err
	}

	return &audience, nil
}

func (r *AudienceRepository) DeleteByID(ctx context.Context, id string) error {

	rowsAffected, err := gorm.G[domain.Audience](r.db).Where("id = ?", id).Delete(ctx)

	if rowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	if err != nil {
		return err
	}

	return nil
}

func (r *AudienceRepository) SaveAll(ctx context.Context, audience []domain.Audience) error {

	err := gorm.G[domain.Audience](r.db).CreateInBatches(ctx, &audience, 100)

	if err != nil {
		return err
	}

	return nil
}
