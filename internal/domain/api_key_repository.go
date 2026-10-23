package domain

import "context"

type ApiKeyRepository interface {
	FindByKey(ctx context.Context, key string) (*ApiKey, error)
	FindByProjectID(ctx context.Context, projectID string) ([]ApiKey, error)
	Save(ctx context.Context, apiKey *ApiKey) error
	Revoke(ctx context.Context, projectID, id string) error
}
