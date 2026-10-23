package services

import (
	"context"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

type EmailConsumer struct {
	emailRepo domain.EmailRepository
	sender    domain.EmailSender
	wait      func(context.Context, time.Duration) error
}

func NewEmailConsumer(emailRepo domain.EmailRepository, sender domain.EmailSender) *EmailConsumer {
	return &EmailConsumer{emailRepo: emailRepo, sender: sender, wait: waitForRetry}
}

func (s *EmailConsumer) Process(ctx context.Context, emailID string) error {
	email, err := s.emailRepo.FindByID(ctx, emailID)
	if err != nil {
		return err
	}
	if email.Status == domain.StatusSent || email.Status == domain.StatusFailed {
		return nil
	}
	delays := []time.Duration{5 * time.Second, 15 * time.Second, 45 * time.Second}
	for email.Attempts < 4 {
		if email.NextAttemptAt != nil {
			if err := s.wait(ctx, time.Until(*email.NextAttemptAt)); err != nil {
				return err
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		email.Status = domain.StatusProcessing
		email.Attempts++
		email.NextAttemptAt = nil
		if err := s.emailRepo.Update(ctx, email); err != nil {
			return err
		}

		sendErr := s.sender.Send(ctx, email)

		if sendErr == nil {
			now := time.Now().UTC()
			email.Status, email.SentAt, email.LastError = domain.StatusSent, &now, ""
			return s.emailRepo.Update(ctx, email)
		}
		email.LastError = sendErr.Error()
		if email.Attempts == 4 {
			break
		}
		next := time.Now().UTC().Add(delays[email.Attempts-1])
		email.NextAttemptAt = &next
		if err := s.emailRepo.Update(ctx, email); err != nil {
			return err
		}
	}
	email.Status, email.NextAttemptAt = domain.StatusFailed, nil
	return s.emailRepo.Update(ctx, email)
}

func waitForRetry(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
