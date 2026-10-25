package domain

import "context"

type EmailRepository interface {
	Save(ctx context.Context, email *Email) error
	SaveAll(ctx context.Context, emails []Email) error
	FindByID(ctx context.Context, id string) (*Email, error)
	FindByProjectID(ctx context.Context, projectID string, status Status, limit int) ([]Email, error)
	CountByStatus(ctx context.Context, projectID string) (map[Status]int64, error)
	Update(ctx context.Context, email *Email) error
}
