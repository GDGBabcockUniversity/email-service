package services

import (
	"context"
	"errors"
	"testing"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSendEmail_ValidRequest_PersistsBeforePublishing(t *testing.T) {
	projectRepo := new(mocks.ProjectRepository)
	templateRepo := new(mocks.TemplateRepository)
	emailRepo := new(mocks.EmailRepository)
	audienceRepo := new(mocks.AudienceRepository)

	publisher := new(mocks.Publisher)

	projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)
	templateRepo.On("FindByNameAndProjectID", mock.Anything, "welcome", "project").Return(&domain.Template{Id: "template", Name: "welcome", ProjectID: "project", Subject: "Hello {{ .name }}", Content: "<p>{{ .name }}</p>"}, nil)

	audienceRepo.On("FindByEmailAndProjectID", mock.Anything, "alice@example.com", "project").Return(nil, domain.ErrNotFound)

	saved := false
	emailRepo.On("Save", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		value := args.Get(1).(*domain.Email)
		assert.Equal(t, domain.StatusPending, value.Status)
		assert.Equal(t, "Hello Alice", value.Subject)
		value.ID = "email"
		saved = true
	}).Return(nil).Once()

	publisher.On("Publish", mock.Anything, "email").Run(func(mock.Arguments) { assert.True(t, saved) }).Return(nil).Once()

	service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher, NewUnsubscribeTokens("unsub-secret", "http://localhost:8080"))
	emails, err := service.SendEmail(context.Background(), "project", &dto.SendEmailRequest{To: "alice@example.com", Template: "welcome", Data: map[string]string{"name": "Alice"}})
	require.NoError(t, err)
	assert.Equal(t, "email", emails[0].ID)
	emailRepo.AssertExpectations(t)
	audienceRepo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

func TestSendEmail_PersistenceOrQueueFailure_ReturnsError(t *testing.T) {
	for _, databaseFails := range []bool{true, false} {
		t.Run(map[bool]string{true: "database", false: "queue"}[databaseFails], func(t *testing.T) {
			projectRepo := new(mocks.ProjectRepository)
			templateRepo := new(mocks.TemplateRepository)
			emailRepo := new(mocks.EmailRepository)
			audienceRepo := new(mocks.AudienceRepository)
			publisher := new(mocks.Publisher)

			projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)
			templateRepo.On("FindByNameAndProjectID", mock.Anything, "welcome", "project").Return(&domain.Template{Id: "template", Name: "welcome", ProjectID: "project", Subject: "Hello", Content: "Hello"}, nil)

			audienceRepo.On("FindByEmailAndProjectID", mock.Anything, "alice@example.com", "project").Return(nil, domain.ErrNotFound)

			var saveErr error
			if databaseFails {
				saveErr = domain.ErrUnavailable
			}
			emailRepo.On("Save", mock.Anything, mock.Anything).Return(saveErr)

			if !databaseFails {
				publisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("redis offline"))
				emailRepo.On("Update", mock.Anything, mock.MatchedBy(func(value *domain.Email) bool { return value.Status == domain.StatusFailed })).Return(nil).Once()
			}
			service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher, NewUnsubscribeTokens("unsub-secret", "http://localhost:8080"))
			email, err := service.SendEmail(context.Background(), "project", &dto.SendEmailRequest{To: "alice@example.com", Template: "welcome"})
			assert.Nil(t, email)
			assert.ErrorIs(t, err, domain.ErrUnavailable)
			if databaseFails {
				publisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
			}
			emailRepo.AssertExpectations(t)
			audienceRepo.AssertExpectations(t)
			publisher.AssertExpectations(t)
		})
	}
}

// Template names are only unique per project, so a send must resolve the name
// inside the authenticated key's project and never reach another project's template.
func TestSendEmail_ResolvesTemplateNameWithinTheRequestingProject(t *testing.T) {
	templateRepo := new(mocks.TemplateRepository)
	emailRepo := new(mocks.EmailRepository)
	audienceRepo := new(mocks.AudienceRepository)
	publisher := new(mocks.Publisher)

	// Both projects own a template called "welcome"; only the caller's may be used.
	templateRepo.On("FindByNameAndProjectID", mock.Anything, "welcome", "project-a").
		Return(&domain.Template{Id: "a", Name: "welcome", ProjectID: "project-a", Subject: "From A", Content: "A"}, nil)
	templateRepo.On("FindByNameAndProjectID", mock.Anything, "welcome", "project-b").
		Return(nil, domain.ErrNotFound)

	audienceRepo.On("FindByEmailAndProjectID", mock.Anything, "alice@example.com", "project-a").Return(nil, domain.ErrNotFound)
	audienceRepo.On("FindByEmailAndProjectID", mock.Anything, "alice@example.com", "project-b").Return(nil, domain.ErrNotFound)
	emailRepo.On("Save", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		args.Get(1).(*domain.Email).ID = "email"
	}).Return(nil).Once()
	publisher.On("Publish", mock.Anything, "email").Return(nil).Once()

	service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher, NewUnsubscribeTokens("unsub-secret", "http://localhost:8080"))
	request := &dto.SendEmailRequest{To: "alice@example.com", Template: "welcome"}

	emails, err := service.SendEmail(context.Background(), "project-a", request)
	require.NoError(t, err)
	assert.Equal(t, "From A", emails[0].Subject)

	// A project without that template name gets a 404, not someone else's template.
	_, err = service.SendEmail(context.Background(), "project-b", request)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	templateRepo.AssertExpectations(t)
	emailRepo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

// A send to the whole audience must skip the recipients who opted out and give
// each remaining recipient their own unsubscribe link.
func TestSendEmail_ToAll_SkipsUnsubscribedAndEmbedsEachRecipientsLink(t *testing.T) {
	templateRepo := new(mocks.TemplateRepository)
	emailRepo := new(mocks.EmailRepository)
	audienceRepo := new(mocks.AudienceRepository)
	publisher := new(mocks.Publisher)
	tokens := NewUnsubscribeTokens("unsub-secret", "https://mail.example.com")

	templateRepo.On("FindByNameAndProjectID", mock.Anything, "news", "project").
		Return(&domain.Template{Id: "template", Name: "news", ProjectID: "project", Subject: "News", Content: `<a href="{{ .UnsubscribeURL }}">Unsubscribe</a>`}, nil)

	// The repository is asked for subscribed recipients only, so Carol (who
	// unsubscribed) is never returned and never considered here.
	audienceRepo.On("FindSubscribedByProjectID", mock.Anything, "project").Return([]domain.Audience{
		{ID: "a1", Email: "alice@example.com", ProjectID: "project", Subscribed: true, UnsubscribeToken: tokens.HashFor("project", "alice@example.com")},
		{ID: "a2", Email: "bob@example.com", ProjectID: "project", Subscribed: true, UnsubscribeToken: tokens.HashFor("project", "bob@example.com")},
	}, nil).Once()

	emailRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Twice()
	publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Twice()

	service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher, tokens)
	emails, err := service.SendEmail(context.Background(), "project", &dto.SendEmailRequest{To: "all", Template: "news"})

	require.NoError(t, err)
	require.Len(t, emails, 2)
	assert.Contains(t, emails[0].Body, tokens.URL("project", "alice@example.com"))
	assert.Contains(t, emails[1].Body, tokens.URL("project", "bob@example.com"))
	assert.NotContains(t, emails[0].Body, tokens.URL("project", "bob@example.com"))
	// Nothing already stored needs rewriting when the hashes are in place.
	audienceRepo.AssertNotCalled(t, "SetUnsubscribeToken", mock.Anything, mock.Anything, mock.Anything)
	audienceRepo.AssertExpectations(t)
	emailRepo.AssertExpectations(t)
	publisher.AssertExpectations(t)
}

// Rows added before unsubscribe links existed hold no token hash, so the first
// send backfills it; otherwise their link would resolve to nothing.
func TestSendEmail_ToAll_BackfillsAMissingTokenHash(t *testing.T) {
	templateRepo := new(mocks.TemplateRepository)
	emailRepo := new(mocks.EmailRepository)
	audienceRepo := new(mocks.AudienceRepository)
	publisher := new(mocks.Publisher)
	tokens := NewUnsubscribeTokens("unsub-secret", "https://mail.example.com")

	templateRepo.On("FindByNameAndProjectID", mock.Anything, "news", "project").
		Return(&domain.Template{Id: "template", Name: "news", ProjectID: "project", Subject: "News", Content: "Hello"}, nil)
	audienceRepo.On("FindSubscribedByProjectID", mock.Anything, "project").Return([]domain.Audience{
		{ID: "a1", Email: "alice@example.com", ProjectID: "project", Subscribed: true},
	}, nil).Once()
	audienceRepo.On("SetUnsubscribeToken", mock.Anything, "a1", tokens.HashFor("project", "alice@example.com")).Return(nil).Once()
	emailRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
	publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Once()

	service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher, tokens)
	_, err := service.SendEmail(context.Background(), "project", &dto.SendEmailRequest{To: "all", Template: "news"})

	require.NoError(t, err)
	audienceRepo.AssertExpectations(t)
}

// A single-address send must honour the same opt-out, not slip past it.
func TestSendEmail_SingleUnsubscribedRecipient_IsRefused(t *testing.T) {
	templateRepo := new(mocks.TemplateRepository)
	emailRepo := new(mocks.EmailRepository)
	audienceRepo := new(mocks.AudienceRepository)
	publisher := new(mocks.Publisher)
	tokens := NewUnsubscribeTokens("unsub-secret", "https://mail.example.com")

	templateRepo.On("FindByNameAndProjectID", mock.Anything, "welcome", "project").
		Return(&domain.Template{Id: "template", Name: "welcome", ProjectID: "project", Subject: "Hi", Content: "Hi"}, nil)
	audienceRepo.On("FindByEmailAndProjectID", mock.Anything, "alice@example.com", "project").
		Return(&domain.Audience{ID: "a1", Email: "alice@example.com", ProjectID: "project", Subscribed: false}, nil).Once()

	service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher, tokens)
	emails, err := service.SendEmail(context.Background(), "project", &dto.SendEmailRequest{To: "alice@example.com", Template: "welcome"})

	assert.Nil(t, emails)
	assert.ErrorIs(t, err, domain.ErrUnsubscribed)
	emailRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
	publisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
}

// An address outside the audience has no subscription to link to, so the
// template still renders with an empty UnsubscribeURL rather than failing.
func TestSendEmail_AddressOutsideTheAudience_RendersAnEmptyUnsubscribeURL(t *testing.T) {
	templateRepo := new(mocks.TemplateRepository)
	emailRepo := new(mocks.EmailRepository)
	audienceRepo := new(mocks.AudienceRepository)
	publisher := new(mocks.Publisher)
	tokens := NewUnsubscribeTokens("unsub-secret", "https://mail.example.com")

	templateRepo.On("FindByNameAndProjectID", mock.Anything, "receipt", "project").
		Return(&domain.Template{Id: "template", Name: "receipt", ProjectID: "project", Subject: "Receipt", Content: `Paid. {{ .UnsubscribeURL }}`}, nil)
	audienceRepo.On("FindByEmailAndProjectID", mock.Anything, "carol@example.com", "project").Return(nil, domain.ErrNotFound).Once()
	emailRepo.On("Save", mock.Anything, mock.Anything).Return(nil).Once()
	publisher.On("Publish", mock.Anything, mock.Anything).Return(nil).Once()

	service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher, tokens)
	emails, err := service.SendEmail(context.Background(), "project", &dto.SendEmailRequest{To: "carol@example.com", Template: "receipt"})

	require.NoError(t, err)
	assert.Equal(t, "Paid. ", emails[0].Body)
}
