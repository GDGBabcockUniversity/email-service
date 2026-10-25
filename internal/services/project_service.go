package services

import (
	"context"
	"errors"
	"strings"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

type ProjectService struct {
	projectRepo  domain.ProjectRepository
	audienceRepo domain.AudienceRepository
	tokens       *UnsubscribeTokens
	ownerID      string
}

func NewProjectService(projectRepo domain.ProjectRepository, audienceRepo domain.AudienceRepository, tokens *UnsubscribeTokens, ownerID string) *ProjectService {
	return &ProjectService{projectRepo: projectRepo, audienceRepo: audienceRepo, tokens: tokens, ownerID: ownerID}
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

// AddAudience subscribes addresses to a project. An address already on the list
// keeps its row: adding it again only turns its subscription back on, so a
// resubscribe never leaves the project with a duplicate audience entry.
func (s *ProjectService) AddAudience(ctx context.Context, projectID string, audience dto.AddAudienceRequest) error {

	var audiences []domain.Audience
	seen := map[string]bool{}

	for _, a := range audience.Audiences {
		email := normalizeEmail(a.Email)
		if seen[email] {
			continue
		}
		seen[email] = true

		existing, err := s.audienceRepo.FindByEmailAndProjectID(ctx, email, projectID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if existing != nil {
			if !existing.Subscribed {
				if err := s.audienceRepo.SetSubscribed(ctx, existing.ID, true); err != nil {
					return err
				}
			}
			continue
		}

		audiences = append(audiences, domain.Audience{
			Email:            email,
			ProjectID:        projectID,
			Subscribed:       true,
			UnsubscribeToken: s.tokens.HashFor(projectID, email),
		})
	}

	if len(audiences) == 0 {
		return nil
	}

	if err := s.audienceRepo.SaveAll(ctx, audiences); err != nil {
		return err
	}

	return nil
}

func (s *ProjectService) ListAudience(ctx context.Context, projectID string) ([]domain.Audience, error) {
	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return nil, err
	}
	return s.audienceRepo.FindByProjectID(ctx, projectID)
}

func (s *ProjectService) RemoveAudience(ctx context.Context, projectID, audienceID string) error {
	audience, err := s.audienceRepo.FindByID(ctx, audienceID)
	if err != nil {
		return err
	}
	if audience == nil || audience.ProjectID != projectID {
		return domain.ErrNotFound
	}
	return s.audienceRepo.DeleteByID(ctx, audienceID)
}
