package domain

import "context"

type AudienceRepository interface {
	FindByProjectID(ctx context.Context, projectID string) ([]Audience, error)
	FindByID(ctx context.Context, id string) (*Audience, error)
	DeleteByID(ctx context.Context, id string) error
	SaveAll(ctx context.Context, audiences []Audience) error
}
