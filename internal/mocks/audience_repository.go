package mocks

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/stretchr/testify/mock"
)

type AudienceRepository struct {
	mock.Mock
}

func (m *AudienceRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.Audience, error) {
	args := m.Called(ctx, projectID)
	var audiences []domain.Audience
	if value := args.Get(0); value != nil {
		audiences = value.([]domain.Audience)
	}
	return audiences, args.Error(1)
}

func (m *AudienceRepository) FindByID(ctx context.Context, id string) (*domain.Audience, error) {
	args := m.Called(ctx, id)
	var audience *domain.Audience
	if value := args.Get(0); value != nil {
		audience = value.(*domain.Audience)
	}
	return audience, args.Error(1)
}

func (m *AudienceRepository) DeleteByID(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *AudienceRepository) SaveAll(ctx context.Context, audiences []domain.Audience) error {
	return m.Called(ctx, audiences).Error(0)
}
