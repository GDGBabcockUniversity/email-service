package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProcess_TransientFailures_RetriesThenMarksSent(t *testing.T) {
	repo := new(mocks.EmailRepository)
	sender := new(mocks.EmailSender)
	email := &domain.Email{ID: "email", Status: domain.StatusPending}
	repo.On("FindByID", mock.Anything, "email").Return(email, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)
	sender.On("Send", mock.Anything, mock.Anything).Return(errors.New("temporary")).Twice()
	sender.On("Send", mock.Anything, mock.Anything).Return(nil).Once()
	consumer := NewEmailConsumer(repo, sender)
	var delays []time.Duration
	consumer.wait = func(_ context.Context, duration time.Duration) error { delays = append(delays, duration); return nil }
	require.NoError(t, consumer.Process(context.Background(), "email"))
	assert.Equal(t, domain.StatusSent, email.Status)
	assert.Equal(t, 3, email.Attempts)
	assert.NotNil(t, email.SentAt)
	assert.Empty(t, email.LastError)
	require.Len(t, delays, 2)
	assert.InDelta(t, 5, delays[0].Seconds(), .2)
	assert.InDelta(t, 15, delays[1].Seconds(), .2)
	sender.AssertExpectations(t)
}

func TestProcess_FourFailures_MarksFailed(t *testing.T) {
	repo := new(mocks.EmailRepository)
	sender := new(mocks.EmailSender)
	email := &domain.Email{ID: "email", Status: domain.StatusPending}
	repo.On("FindByID", mock.Anything, "email").Return(email, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)
	sender.On("Send", mock.Anything, mock.Anything).Return(errors.New("SMTP rejected")).Times(4)
	consumer := NewEmailConsumer(repo, sender)
	var delays []time.Duration
	consumer.wait = func(_ context.Context, duration time.Duration) error { delays = append(delays, duration); return nil }
	require.NoError(t, consumer.Process(context.Background(), "email"))
	assert.Equal(t, domain.StatusFailed, email.Status)
	assert.Equal(t, 4, email.Attempts)
	assert.Nil(t, email.SentAt)
	assert.Nil(t, email.NextAttemptAt)
	require.Len(t, delays, 3)
	assert.InDelta(t, 45, delays[2].Seconds(), .2)
	sender.AssertExpectations(t)
}

func TestProcess_TerminalEmail_DoesNotSendAgain(t *testing.T) {
	for _, status := range []domain.Status{domain.StatusSent, domain.StatusFailed} {
		repo := new(mocks.EmailRepository)
		sender := new(mocks.EmailSender)
		repo.On("FindByID", mock.Anything, "email").Return(&domain.Email{Status: status}, nil)
		require.NoError(t, NewEmailConsumer(repo, sender).Process(context.Background(), "email"))
		sender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
		repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	}
}

func TestProcess_ResumedEmail_PreservesRetryBudget(t *testing.T) {
	repo := new(mocks.EmailRepository)
	sender := new(mocks.EmailSender)
	next := time.Now().Add(15 * time.Second)
	email := &domain.Email{ID: "email", Status: domain.StatusProcessing, Attempts: 2, NextAttemptAt: &next}
	repo.On("FindByID", mock.Anything, "email").Return(email, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)
	sender.On("Send", mock.Anything, mock.Anything).Return(errors.New("SMTP rejected")).Twice()
	consumer := NewEmailConsumer(repo, sender)
	waits := 0
	consumer.wait = func(context.Context, time.Duration) error { waits++; return nil }
	require.NoError(t, consumer.Process(context.Background(), "email"))
	assert.Equal(t, 4, email.Attempts)
	assert.Equal(t, 2, waits)
	assert.Equal(t, domain.StatusFailed, email.Status)
	sender.AssertExpectations(t)
}

func TestProcess_PersistenceFailure_DoesNotSend(t *testing.T) {
	repo := new(mocks.EmailRepository)
	sender := new(mocks.EmailSender)
	repo.On("FindByID", mock.Anything, "email").Return(&domain.Email{}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(domain.ErrUnavailable)
	assert.ErrorIs(t, NewEmailConsumer(repo, sender).Process(context.Background(), "email"), domain.ErrUnavailable)
	sender.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}

func TestProcess_FinalPersistenceFailure_LeavesJobUnacknowledged(t *testing.T) {
	repo := new(mocks.EmailRepository)
	sender := new(mocks.EmailSender)
	repo.On("FindByID", mock.Anything, "email").Return(&domain.Email{}, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(email *domain.Email) bool { return email.Status == domain.StatusProcessing })).Return(nil).Once()
	repo.On("Update", mock.Anything, mock.MatchedBy(func(email *domain.Email) bool { return email.Status == domain.StatusSent })).Return(domain.ErrUnavailable).Once()
	sender.On("Send", mock.Anything, mock.Anything).Return(nil).Once()
	assert.ErrorIs(t, NewEmailConsumer(repo, sender).Process(context.Background(), "email"), domain.ErrUnavailable)
}

func TestWaitForRetry_CancelledContext_ReturnsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.ErrorIs(t, waitForRetry(ctx, time.Hour), context.Canceled)
}
