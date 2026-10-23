package domain

import "context"

type ProjectRepository interface {
	Update(ctx context.Context, project *Project) error
	FindByID(ctx context.Context, id string) (*Project, error)
	FindAll(ctx context.Context) ([]Project, error)
	Save(ctx context.Context, project *Project) error
	DeleteById(ctx context.Context, id string) error
	ExistsByNameAndUserID(ctx context.Context, name, userID string) (bool, error)
	ExistsByID(ctx context.Context, id string) (bool, error)
	AddAudience(ctx context.Context, projectID string, audience []Audience) error
}
