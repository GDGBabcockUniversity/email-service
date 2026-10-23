package mocks

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/stretchr/testify/mock"
)

type ApiKeyRepository struct{ mock.Mock }

func (m *ApiKeyRepository) FindByKey(ctx context.Context, key string) (*domain.ApiKey, error) {
	args := m.Called(ctx, key)
	var apiKey *domain.ApiKey
	if value := args.Get(0); value != nil {
		apiKey = value.(*domain.ApiKey)
	}
	return apiKey, args.Error(1)
}

func (m *ApiKeyRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.ApiKey, error) {
	args := m.Called(ctx, projectID)
	var keys []domain.ApiKey
	if value := args.Get(0); value != nil {
		keys = value.([]domain.ApiKey)
	}
	return keys, args.Error(1)
}

func (m *ApiKeyRepository) Save(ctx context.Context, apiKey *domain.ApiKey) error {
	return m.Called(ctx, apiKey).Error(0)
}

func (m *ApiKeyRepository) Revoke(ctx context.Context, projectID, id string) error {
	return m.Called(ctx, projectID, id).Error(0)
}
