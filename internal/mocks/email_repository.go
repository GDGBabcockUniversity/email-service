package mocks

import (
	"context"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/stretchr/testify/mock"
)

type EmailRepository struct{ mock.Mock }

func (m *EmailRepository) Save(ctx context.Context, email *domain.Email) error {
	return m.Called(ctx, email).Error(0)
}

func (m *EmailRepository) Update(ctx context.Context, email *domain.Email) error {
	return m.Called(ctx, email).Error(0)
}

func (m *EmailRepository) FindByID(ctx context.Context, id string) (*domain.Email, error) {
	args := m.Called(ctx, id)
	var email *domain.Email
	if value := args.Get(0); value != nil {
		email = value.(*domain.Email)
	}
	return email, args.Error(1)
}

func (m *EmailRepository) SaveAll(ctx context.Context, emails []domain.Email) error {
	return m.Called(ctx, emails).Error(0)
}

func (m *EmailRepository) FindByProjectID(ctx context.Context, projectID string, status domain.Status, limit int) ([]domain.Email, error) {
	args := m.Called(ctx, projectID, status, limit)
	var emails []domain.Email
	if value := args.Get(0); value != nil {
		emails = value.([]domain.Email)
	}
	return emails, args.Error(1)
}

func (m *EmailRepository) CountByStatus(ctx context.Context, projectID string) (map[domain.Status]int64, error) {
	args := m.Called(ctx, projectID)
	var counts map[domain.Status]int64
	if value := args.Get(0); value != nil {
		counts = value.(map[domain.Status]int64)
	}
	return counts, args.Error(1)
}

type Publisher struct{ mock.Mock }

func (m *Publisher) Publish(ctx context.Context, emailID string) error {
	return m.Called(ctx, emailID).Error(0)
}

type EmailSender struct{ mock.Mock }

func (m *EmailSender) Send(ctx context.Context, email *domain.Email) error {
	return m.Called(ctx, email).Error(0)
}
