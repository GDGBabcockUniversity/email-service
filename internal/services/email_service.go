package services

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

type EmailService struct {
	emailRepo       domain.EmailRepository
	audienceRepo    domain.AudienceRepository
	templateRepo domain.TemplateRepository
	publisher       domain.Publisher
}

func NewEmailService(emailRepo domain.EmailRepository, audienceRepo domain.AudienceRepository, templateRepo domain.TemplateRepository, publisher domain.Publisher) *EmailService {
	return &EmailService{emailRepo: emailRepo, audienceRepo: audienceRepo, templateRepo: templateRepo, publisher: publisher}
}

func (s *EmailService) SendEmail(ctx context.Context, projectID string, request *dto.SendEmailRequest) ([]*domain.Email, error) {

	if request.To != "all" {
		address, err := mail.ParseAddress(request.To)
	
		if err != nil || address.Address != request.To || strings.ContainsAny(request.To, "\r\n") {
			return nil, domain.ErrInvalidRequest
		}
	}
	
	record, err := s.templateRepo.FindByNameAndProjectID(ctx, request.Template, projectID)
	if err != nil {
		return nil, err
	}

	if record == nil {
		return nil, domain.ErrNotFound
	}
	
	subject, body, err := renderTemplate(record, request.Data)
	if err != nil {
		return nil, err
	}

	audiences, err := s.audienceRepo.FindByProjectID(ctx, projectID)

	slog.Info("audiences", "AUD", audiences)

	if err != nil {
		return nil, err
	}

	if request.To == "all" {

		var emails []*domain.Email
		for _, audience := range audiences {

			email := &domain.Email{
				To:         audience.Email,
				ProjectID:  projectID,
				TemplateID: record.Id,
				Subject:    subject,
				Body:       body,
				Status:     domain.StatusPending,
			}

			emails = append(emails, email)

			if err := s.emailRepo.Save(ctx, email); err != nil {
				return nil, err
			}
			if err := s.publisher.Publish(ctx, email.ID); err != nil {
				email.Status, email.LastError = domain.StatusFailed, "Queue publication failed"
				// Persist failure even when the requesting client has disconnected.
				failureCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				if updateErr := s.emailRepo.Update(failureCtx, email); updateErr != nil {
					return nil, fmt.Errorf("%w: queue publication and failure persistence failed", domain.ErrUnavailable)
				}
				return nil, domain.ErrUnavailable
			}

		}

		return emails, nil

	}
	email := &domain.Email{
		To:         request.To,
		ProjectID:  projectID,
		TemplateID: record.Id,
		Subject:    subject,
		Body:       body,
		Status:     domain.StatusPending,
	}

	if err := s.emailRepo.Save(ctx, email); err != nil {
		return nil, err
	}
	if err := s.publisher.Publish(ctx, email.ID); err != nil {
		email.Status, email.LastError = domain.StatusFailed, "Queue publication failed"
		// Persist failure even when the requesting client has disconnected.
		failureCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if updateErr := s.emailRepo.Update(failureCtx, email); updateErr != nil {
			return nil, fmt.Errorf("%w: queue publication and failure persistence failed", domain.ErrUnavailable)
		}
		return nil, domain.ErrUnavailable
	}
	return []*domain.Email{email}, nil
}
// ListEmails returns a project's most recent emails for the admin dashboard,
// optionally narrowed to one delivery status.
func (s *EmailService) ListEmails(ctx context.Context, projectID, status string, limit int) ([]domain.Email, error) {
	filter := domain.Status(strings.ToUpper(strings.TrimSpace(status)))
	switch filter {
	case "", domain.StatusPending, domain.StatusProcessing, domain.StatusSent, domain.StatusFailed:
	default:
		return nil, domain.ErrInvalidRequest
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return s.emailRepo.FindByProjectID(ctx, projectID, filter, limit)
}

// EmailStats tallies a project's delivery outcomes, always reporting every status.
func (s *EmailService) EmailStats(ctx context.Context, projectID string) (*dto.EmailStats, error) {
	counts, err := s.emailRepo.CountByStatus(ctx, projectID)
	if err != nil {
		return nil, err
	}
	stats := &dto.EmailStats{
		Pending:    counts[domain.StatusPending],
		Processing: counts[domain.StatusProcessing],
		Sent:       counts[domain.StatusSent],
		Failed:     counts[domain.StatusFailed],
	}
	stats.Total = stats.Pending + stats.Processing + stats.Sent + stats.Failed
	return stats, nil
}
