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

	audienceRepo.On("FindByProjectID", mock.Anything, "project").Return([]domain.Audience{}, nil)

	saved := false
	emailRepo.On("Save", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		value := args.Get(1).(*domain.Email)
		assert.Equal(t, domain.StatusPending, value.Status)
		assert.Equal(t, "Hello Alice", value.Subject)
		value.ID = "email"
		saved = true
	}).Return(nil).Once()

	publisher.On("Publish", mock.Anything, "email").Run(func(mock.Arguments) { assert.True(t, saved) }).Return(nil).Once()

	service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher)
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

			audienceRepo.On("FindByProjectID", mock.Anything, "project").Return([]domain.Audience{}, nil)

			var saveErr error
			if databaseFails {
				saveErr = domain.ErrUnavailable
			}
			emailRepo.On("Save", mock.Anything, mock.Anything).Return(saveErr)

			if !databaseFails {
				publisher.On("Publish", mock.Anything, mock.Anything).Return(errors.New("redis offline"))
				emailRepo.On("Update", mock.Anything, mock.MatchedBy(func(value *domain.Email) bool { return value.Status == domain.StatusFailed })).Return(nil).Once()
			}
			service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher)
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

	audienceRepo.On("FindByProjectID", mock.Anything, "project-a").Return([]domain.Audience{}, nil)
	emailRepo.On("Save", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		args.Get(1).(*domain.Email).ID = "email"
	}).Return(nil).Once()
	publisher.On("Publish", mock.Anything, "email").Return(nil).Once()

	service := NewEmailService(emailRepo, audienceRepo, templateRepo, publisher)
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
