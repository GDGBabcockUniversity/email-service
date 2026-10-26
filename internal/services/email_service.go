package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/mail"
	"strings"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

type EmailService struct {
	emailRepo    domain.EmailRepository
	audienceRepo domain.AudienceRepository
	templateRepo domain.TemplateRepository
	publisher    domain.Publisher
	tokens       *UnsubscribeTokens
}

func NewEmailService(emailRepo domain.EmailRepository, audienceRepo domain.AudienceRepository, templateRepo domain.TemplateRepository, publisher domain.Publisher, tokens *UnsubscribeTokens) *EmailService {
	return &EmailService{emailRepo: emailRepo, audienceRepo: audienceRepo, templateRepo: templateRepo, publisher: publisher, tokens: tokens}
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

	if request.To == "all" {

		// Only recipients who still consent; unsubscribed rows stay in the
		// audience but are never sent to again.
		audiences, err := s.audienceRepo.FindSubscribedByProjectID(ctx, projectID)
		if err != nil {
			return nil, err
		}

		slog.Info("audiences", "AUD", audiences)

		emails := []*domain.Email{}
		for _, audience := range audiences {

			if err := s.ensureUnsubscribeToken(ctx, &audience); err != nil {
				return nil, err
			}

			// Each recipient's body carries their own unsubscribe link, so the
			// template has to be rendered per address rather than once.
			subject, body, err := renderTemplate(record, s.recipientData(request.Data, projectID, audience.Email))
			if err != nil {
				return nil, err
			}

			email := &domain.Email{
				To:         audience.Email,
				ProjectID:  projectID,
				TemplateID: record.Id,
				Subject:    subject,
				Body:       body,
				Status:     domain.StatusPending,
			}

			emails = append(emails, email)

			if err := s.queue(ctx, email); err != nil {
				return nil, err
			}

		}

		return emails, nil

	}

	// A one-off send still has to honour an unsubscribe: if this address is on
	// the project's audience and has opted out, skip it rather than deliver.
	audience, err := s.audienceRepo.FindByEmailAndProjectID(ctx, request.To, projectID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if audience != nil {
		// Honouring the opt-out is the request succeeding, not failing: the
		// caller is told nothing was queued and has nothing to retry. This
		// matches the to: "all" path, which has always skipped these silently.
		if !audience.Subscribed {
			return []*domain.Email{}, nil
		}
		if err := s.ensureUnsubscribeToken(ctx, audience); err != nil {
			return nil, err
		}
	} else if err := s.enrollRecipient(ctx, projectID, request.To); err != nil {
		return nil, err
	}

	subject, body, err := renderTemplate(record, s.recipientData(request.Data, projectID, request.To))
	if err != nil {
		return nil, err
	}

	email := &domain.Email{
		To:         request.To,
		ProjectID:  projectID,
		TemplateID: record.Id,
		Subject:    subject,
		Body:       body,
		Status:     domain.StatusPending,
	}

	if err := s.queue(ctx, email); err != nil {
		return nil, err
	}
	return []*domain.Email{email}, nil
}

// queue persists an email and publishes it, marking the row failed if the queue
// rejects it so a dropped job is never silently reported as pending.
func (s *EmailService) queue(ctx context.Context, email *domain.Email) error {
	if err := s.emailRepo.Save(ctx, email); err != nil {
		return err
	}
	if err := s.publisher.Publish(ctx, email.ID); err != nil {
		email.Status, email.LastError = domain.StatusFailed, "Queue publication failed"
		// Persist failure even when the requesting client has disconnected.
		failureCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if updateErr := s.emailRepo.Update(failureCtx, email); updateErr != nil {
			return fmt.Errorf("%w: queue publication and failure persistence failed", domain.ErrUnavailable)
		}
		return domain.ErrUnavailable
	}
	return nil
}

// recipientData copies the caller's template data and adds the recipient's
// unsubscribe link, so templates can use {{ .UnsubscribeURL }}. The key is
// always present (empty for an address that holds no subscription) because
// templates render with missingkey=error.
func (s *EmailService) recipientData(data map[string]string, projectID, email string) map[string]string {
	merged := make(map[string]string, len(data)+1)
	maps.Copy(merged, data)

	link := ""
	if email != "" && s.tokens != nil {
		link = s.tokens.URL(projectID, email)
	}

	merged["UnsubscribeURL"] = link
	return merged
}

// ensureUnsubscribeToken backfills the stored token hash for subscriptions added
// before unsubscribe links existed, so their links resolve on the first click.
func (s *EmailService) ensureUnsubscribeToken(ctx context.Context, audience *domain.Audience) error {
	if s.tokens == nil {
		return nil
	}
	expected := s.tokens.HashFor(audience.ProjectID, audience.Email)
	if audience.UnsubscribeToken == expected {
		return nil
	}
	if err := s.audienceRepo.SetUnsubscribeToken(ctx, audience.ID, expected); err != nil {
		return err
	}
	audience.UnsubscribeToken = expected
	return nil
}

// enrollRecipient records a one-off recipient on the project's audience so the
// unsubscribe link in their email resolves to a subscription. Without a row the
// token has nothing to point at, and the link would have to be sent without it.
func (s *EmailService) enrollRecipient(ctx context.Context, projectID, email string) error {
	if s.tokens == nil {
		return nil
	}
	return s.audienceRepo.SaveAll(ctx, []domain.Audience{{
		Email:            email,
		ProjectID:        projectID,
		Subscribed:       true,
		UnsubscribeToken: s.tokens.HashFor(projectID, email),
	}})
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
