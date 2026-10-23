package services

import (
	"context"
	"testing"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateProject_ValidRequest_ReturnsNoError(t *testing.T) {

	projectRepo := new(mocks.ProjectRepository)

	audienceRepo := new(mocks.AudienceRepository)

	request := dto.CreateProjectRequest{
		Name:        "project-1",
		Description: "mock project",
	}

	project := domain.Project{
		Name:        request.Name,
		Description: request.Description,
		UserID:      "owner-id",
	}

	projectRepo.On("ExistsByNameAndUserID", context.Background(), request.Name, mock.Anything).Return(false, nil)

	projectRepo.On("Save", context.Background(), &project).Return(nil)

	projectService := NewProjectService(projectRepo, audienceRepo, "owner-id")

	_, err := projectService.CreateProject(context.Background(), &request)

	projectRepo.AssertCalled(t, "ExistsByNameAndUserID", context.Background(), request.Name, mock.Anything)
	projectRepo.AssertCalled(t, "Save", context.Background(), &project)
	projectRepo.AssertNumberOfCalls(t, "ExistsByNameAndUserID", 1)
	projectRepo.AssertNumberOfCalls(t, "Save", 1)

	assert.NoError(t, err)
}

func TestCreateProject_ExistingProjectName_ThrowsDuplicateProjectError(t *testing.T) {

	projectRepo := new(mocks.ProjectRepository)

	audienceRepo := new(mocks.AudienceRepository)

	request := dto.CreateProjectRequest{
		Name:        "project-1",
		Description: "mock project",
	}

	projectRepo.On("ExistsByNameAndUserID", context.Background(), request.Name, mock.Anything).Return(true, nil)

	projectService := NewProjectService(projectRepo, audienceRepo, "owner-id")

	_, err := projectService.CreateProject(context.Background(), &request)

	projectRepo.AssertNotCalled(t, "Save")

	assert.ErrorIs(t, err, domain.ErrDuplicateProjectName)
}
