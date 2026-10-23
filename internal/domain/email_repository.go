package domain

import "context"

type EmailRepository interface {
	Save(ctx context.Context, email *Email) error
	SaveAll(ctx context.Context, emails []Email) error
	FindByID(ctx context.Context, id string) (*Email, error)
	Update(ctx context.Context, email *Email) error
}
