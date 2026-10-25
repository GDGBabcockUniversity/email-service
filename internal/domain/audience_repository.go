package domain

import "context"

type AudienceRepository interface {
	FindByProjectID(ctx context.Context, projectID string) ([]Audience, error)
	FindSubscribedByProjectID(ctx context.Context, projectID string) ([]Audience, error)
	FindByID(ctx context.Context, id string) (*Audience, error)
	FindByEmailAndProjectID(ctx context.Context, email, projectID string) (*Audience, error)
	FindByUnsubscribeToken(ctx context.Context, tokenHash string) (*Audience, error)
	DeleteByID(ctx context.Context, id string) error
	SaveAll(ctx context.Context, audiences []Audience) error
	SetSubscribed(ctx context.Context, id string, subscribed bool) error
	SetUnsubscribeToken(ctx context.Context, id, tokenHash string) error
}
