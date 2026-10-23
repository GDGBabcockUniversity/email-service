package services

import (
	"context"
	"strings"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

type ProjectService struct {
	projectRepo  domain.ProjectRepository
	audienceRepo domain.AudienceRepository
	ownerID      string
}

func NewProjectService(projectRepo domain.ProjectRepository, audienceRepo domain.AudienceRepository, ownerID string) *ProjectService {
	return &ProjectService{projectRepo: projectRepo, audienceRepo: audienceRepo, ownerID: ownerID}
}

func (s *ProjectService) CreateProject(ctx context.Context, request *dto.CreateProjectRequest) (*domain.Project, error) {
	if strings.TrimSpace(request.Name) == "" {
		return nil, domain.ErrInvalidRequest
	}
	exists, err := s.projectRepo.ExistsByNameAndUserID(ctx, request.Name, s.ownerID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrDuplicateProjectName
	}
	project := &domain.Project{Name: request.Name, Description: request.Description, UserID: s.ownerID}
	if err := s.projectRepo.Save(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) FindAll(ctx context.Context) ([]domain.Project, error) {
	return s.projectRepo.FindAll(ctx)
}

func (s *ProjectService) FindByID(ctx context.Context, id string) (*domain.Project, error) {
	return s.projectRepo.FindByID(ctx, id)
}

func (s *ProjectService) UpdateProject(ctx context.Context, id string, request *dto.CreateProjectRequest) (*domain.Project, error) {
	if strings.TrimSpace(request.Name) == "" {
		return nil, domain.ErrInvalidRequest
	}
	project, err := s.projectRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	project.Name, project.Description = request.Name, request.Description
	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, id string) error {
	return s.projectRepo.DeleteById(ctx, id)
}

func (s *ProjectService) AddAudience(ctx context.Context, projectID string, audience dto.AddAudienceRequest) error {

	var audiences []domain.Audience

	for _, a := range audience.Audiences {
		audiences = append(audiences, domain.Audience{Email: a.Email, ProjectID: projectID})
	}

	if err := s.audienceRepo.SaveAll(ctx, audiences); err != nil {
		return err
	}

	return nil
}
