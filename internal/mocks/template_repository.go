package mocks

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/stretchr/testify/mock"
)

type TemplateRepository struct {
	mock.Mock
}

func (m *TemplateRepository) FindByID(ctx context.Context, id string) (*domain.Template, error) {

	var template *domain.Template

	args := m.Called(ctx, id)

	if value := args.Get(0); value != nil {
		template = value.(*domain.Template)
	}

	return template, args.Error(1)
}

func (m *TemplateRepository) FindByNameAndProjectID(ctx context.Context, name, projectID string) (*domain.Template, error) {

	var template *domain.Template

	args := m.Called(ctx, name, projectID)

	if value := args.Get(0); value != nil {
		template = value.(*domain.Template)
	}

	return template, args.Error(1)
}

func (m *TemplateRepository) FindAll(ctx context.Context) ([]domain.Template, error) {
	var templates []domain.Template

	args := m.Called(ctx)

	if value := args.Get(0); value != nil {
		templates = value.([]domain.Template)
	}

	return templates, args.Error(1)
}

func (m *TemplateRepository) Save(ctx context.Context, template *domain.Template) error {
	args := m.Called(ctx, template)

	return args.Error(0)
}

func (m *TemplateRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)

	return args.Error(0)
}

func (m *TemplateRepository) Update(ctx context.Context, template *domain.Template) error {
	return m.Called(ctx, template).Error(0)
}

func (m *TemplateRepository) FindByProjectID(ctx context.Context, projectID string) ([]domain.Template, error) {
	args := m.Called(ctx, projectID)
	var templates []domain.Template
	if value := args.Get(0); value != nil {
		templates = value.([]domain.Template)
	}
	return templates, args.Error(1)
}
