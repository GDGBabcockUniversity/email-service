package postgres

import (
	"context"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"gorm.io/gorm"
)

type ApiKeyRepository struct{ db *gorm.DB }

func NewApiKeyRepository(db *gorm.DB) *ApiKeyRepository { return &ApiKeyRepository{db: db} }

func (r *ApiKeyRepository) FindByKey(ctx context.Context, key string) (*domain.ApiKey, error) {
	apiKey, err := gorm.G[domain.ApiKey](r.db).Where("key = ?", key).First(ctx)
	if err != nil {
		return nil, repositoryError(err)
	}
	return &apiKey, nil
}

func (r *ApiKeyRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.ApiKey, error) {
	keys, err := gorm.G[domain.ApiKey](r.db).Where("project_id = ?", projectID).Order("created_at, id").Find(ctx)
	return keys, repositoryError(err)
}

func (r *ApiKeyRepository) Save(ctx context.Context, apiKey *domain.ApiKey) error {
	return repositoryError(gorm.G[domain.ApiKey](r.db).Create(ctx, apiKey))
}

func (r *ApiKeyRepository) Revoke(ctx context.Context, projectID, id string) error {
	result := r.db.WithContext(ctx).Model(&domain.ApiKey{}).Where("id = ? AND project_id = ?", id, projectID).
		Updates(map[string]any{"revoked": true, "revoked_at": time.Now().UTC()})
	return changed(result.RowsAffected, result.Error)
}
