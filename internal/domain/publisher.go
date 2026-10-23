package domain

import "context"

type Publisher interface {
	Publish(ctx context.Context, emailID string) error
}

type EmailSender interface {
	Send(ctx context.Context, email *Email) error
}
