package mocks

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/stretchr/testify/mock"
)

type ProjectRepository struct {
	mock.Mock
}

func (m *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	return m.Called(ctx, project).Error(0)
}

func (m *ProjectRepository) FindByID(ctx context.Context, id string) (*domain.Project, error) {
	args := m.Called(ctx, id)

	var project *domain.Project
	if value := args.Get(0); value != nil {
		project = value.(*domain.Project)
	}

	return project, args.Error(1)
}

func (m *ProjectRepository) FindAll(ctx context.Context) ([]domain.Project, error) {
	args := m.Called(ctx)

	var projects []domain.Project
	if value := args.Get(0); value != nil {
		projects = value.([]domain.Project)
	}

	return projects, args.Error(1)
}

func (m *ProjectRepository) Save(ctx context.Context, project *domain.Project) error {
	args := m.Called(ctx, project)
	return args.Error(0)
}

func (m *ProjectRepository) DeleteById(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *ProjectRepository) ExistsByNameAndUserID(ctx context.Context, name, userID string) (bool, error) {
	args := m.Called(ctx, name, userID)
	return args.Bool(0), args.Error(1)
}

func (m *ProjectRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *ProjectRepository) AddAudience(ctx context.Context, projectID string, audience []domain.Audience) error {
	args := m.Called(ctx, projectID, audience)
	return args.Error(0)
}
