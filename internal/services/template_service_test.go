package services

import (
	"context"
	"testing"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateTemplate_ValidRequest_ReturnNoError(t *testing.T) {
	projectID := "project-id"
	projectRepo := new(mocks.ProjectRepository)
	templateRepo := new(mocks.TemplateRepository)

	request := dto.CreateTemplateRequest{Name: "Welcome", Subject: "Hello {{ .name }}", Content: "<h2>Hello {{ .name }}</h2>"}

	projectRepo.On("FindByID", mock.Anything, projectID).Return(&domain.Project{ID: projectID, UserID: "owner-id"}, nil)

	stored := request.Content + "\n" + unsubscribeFooter

	templateRepo.On("Save", mock.Anything, mock.MatchedBy(func(value *domain.Template) bool {
		return value.Content == stored && value.Subject == request.Subject && value.UserID == "owner-id" && value.ProjectID == projectID && value.Data != nil
	})).Return(nil)

	service := NewTemplateService(projectRepo, templateRepo)

	record, err := service.CreateTemplate(context.Background(), projectID, request)

	require.NoError(t, err)
	assert.Equal(t, stored, record.Content)
	projectRepo.AssertExpectations(t)
	templateRepo.AssertExpectations(t)
}

func TestCreateTemplate_InvalidTemplate_DoesNotSave(t *testing.T) {
	projectRepo := new(mocks.ProjectRepository)
	templateRepo := new(mocks.TemplateRepository)
	projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)
	service := NewTemplateService(projectRepo, templateRepo)

	_, err := service.CreateTemplate(context.Background(), "project", dto.CreateTemplateRequest{Name: "Welcome", Subject: "Hello", Content: "{{"})

	assert.ErrorIs(t, err, domain.ErrInvalidRequest)
	templateRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestFindTemplate_OtherProject_ReturnsNotFound(t *testing.T) {
	projectRepo := new(mocks.ProjectRepository)
	templateRepo := new(mocks.TemplateRepository)

	projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)

	templateRepo.On("FindByID", mock.Anything, "template").Return(&domain.Template{ProjectID: "other"}, nil)

	service := NewTemplateService(projectRepo, templateRepo)

	_, err := service.GetTemplateByID(context.Background(), "project", "template")

	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetTemplateByID_ReturnsTemplate(t *testing.T) {
	projectRepo := new(mocks.ProjectRepository)
	templateRepo := new(mocks.TemplateRepository)
	record := &domain.Template{Id: "template", ProjectID: "project", Name: "Welcome"}
	projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)
	templateRepo.On("FindByID", mock.Anything, "template").Return(record, nil)

	result, err := NewTemplateService(projectRepo, templateRepo).GetTemplateByID(context.Background(), "project", "template")

	require.NoError(t, err)
	assert.Equal(t, record, result)
}

func TestDeleteTemplate_UsesIDAndChecksProject(t *testing.T) {
	for _, projectID := range []string{"project", "other"} {
		t.Run(projectID, func(t *testing.T) {
			projectRepo := new(mocks.ProjectRepository)
			templateRepo := new(mocks.TemplateRepository)
			projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)
			templateRepo.On("FindByID", mock.Anything, "template").Return(&domain.Template{Id: "template", ProjectID: projectID}, nil)
			if projectID == "project" {
				templateRepo.On("Delete", mock.Anything, "template").Return(nil).Once()
			}

			err := NewTemplateService(projectRepo, templateRepo).DeleteTemplate(context.Background(), "project", "template")

			if projectID == "project" {
				require.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, domain.ErrNotFound)
				templateRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			}
			templateRepo.AssertExpectations(t)
		})
	}
}

func TestRenderTemplate_RuntimeDataEscapesHTML(t *testing.T) {
	record := &domain.Template{Subject: "Hello {{ .name }}", Content: "<h2>{{ .name }}</h2>", Data: []string{"name"}}
	subject, body, err := renderTemplate(record, map[string]string{"name": "<Alice>"})
	require.NoError(t, err)
	assert.Equal(t, "Hello <Alice>", subject)
	assert.Equal(t, "<h2>&lt;Alice&gt;</h2>", body)
}

func TestRenderTemplate_MissingVariablesOrHeaderInjection_ReturnsError(t *testing.T) {
	record := &domain.Template{Subject: "Hello {{ .name }}", Content: "<h2>{{ .name }}</h2>"}
	for _, data := range []map[string]string{nil, {"Name": "Alice"}, {"name": "Alice\r\nBcc: another@example.com"}} {
		_, _, err := renderTemplate(record, data)
		assert.ErrorIs(t, err, domain.ErrInvalidRequest)
	}
}

// The unsubscribe footer is the service's job, not the dashboard's, so it has
// to land on templates created or updated straight through the API too.
func TestCreateTemplate_AppendsTheUnsubscribeFooter(t *testing.T) {
	for name, content := range map[string]string{
		"no link":           "<p>Hello.</p>",
		"trailing newlines": "<p>Hello.</p>\n\n  ",
	} {
		t.Run(name, func(t *testing.T) {
			projectRepo := new(mocks.ProjectRepository)
			templateRepo := new(mocks.TemplateRepository)
			projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{ID: "project"}, nil)
			templateRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

			record, err := NewTemplateService(projectRepo, templateRepo).
				CreateTemplate(context.Background(), "project", dto.CreateTemplateRequest{Name: "news", Subject: "News", Content: content})

			require.NoError(t, err)
			assert.Equal(t, "<p>Hello.</p>\n"+unsubscribeFooter, record.Content)
		})
	}
}

// An author may move the link, so a body that already references it is stored
// exactly as submitted rather than collecting a second footer.
func TestCreateTemplate_BodyThatPlacesTheLinkItself_IsLeftAlone(t *testing.T) {
	for _, content := range []string{
		`<a href="{{ .UnsubscribeURL }}">Leave</a><p>Hello.</p>`,
		`<a href="{{.UnsubscribeURL}}">Leave</a><p>Hello.</p>`,
		`<a href="{{- .UnsubscribeURL -}}">Leave</a><p>Hello.</p>`,
	} {
		projectRepo := new(mocks.ProjectRepository)
		templateRepo := new(mocks.TemplateRepository)
		projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{ID: "project"}, nil)
		templateRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

		record, err := NewTemplateService(projectRepo, templateRepo).
			CreateTemplate(context.Background(), "project", dto.CreateTemplateRequest{Name: "news", Subject: "News", Content: content})

		require.NoError(t, err)
		assert.Equal(t, content, record.Content)
	}
}

// Appending must not turn an empty body into a valid one.
func TestCreateTemplate_EmptyBody_IsStillRejected(t *testing.T) {
	projectRepo := new(mocks.ProjectRepository)
	templateRepo := new(mocks.TemplateRepository)
	projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)

	_, err := NewTemplateService(projectRepo, templateRepo).
		CreateTemplate(context.Background(), "project", dto.CreateTemplateRequest{Name: "news", Subject: "News", Content: "   "})

	assert.ErrorIs(t, err, domain.ErrInvalidRequest)
	templateRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestUpdateTemplate_AppendsTheUnsubscribeFooter(t *testing.T) {
	projectRepo := new(mocks.ProjectRepository)
	templateRepo := new(mocks.TemplateRepository)
	projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{ID: "project"}, nil)
	templateRepo.On("FindByID", mock.Anything, "template").
		Return(&domain.Template{Id: "template", ProjectID: "project"}, nil)
	templateRepo.On("Update", mock.Anything, mock.MatchedBy(func(value *domain.Template) bool {
		return value.Content == "<p>Fresh.</p>\n"+unsubscribeFooter
	})).Return(nil)

	record, err := NewTemplateService(projectRepo, templateRepo).
		UpdateTemplate(context.Background(), "project", "template", dto.CreateTemplateRequest{Name: "news", Subject: "News", Content: "<p>Fresh.</p>"})

	require.NoError(t, err)
	assert.Equal(t, "<p>Fresh.</p>\n"+unsubscribeFooter, record.Content)
	templateRepo.AssertExpectations(t)
}
