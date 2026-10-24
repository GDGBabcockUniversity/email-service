package domain

import "context"

type TemplateRepository interface {
	Update(ctx context.Context, template *Template) error
	FindByProjectID(ctx context.Context, projectID string) ([]Template, error)
	FindByID(ctx context.Context, id string) (*Template, error)
	FindByNameAndProjectID(ctx context.Context, name, projectID string) (*Template, error)
	FindAll(ctx context.Context) ([]Template, error)
	Save(ctx context.Context, template *Template) error
	Delete(ctx context.Context, id string) error
}
