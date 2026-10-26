package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	texttemplate "text/template"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

type TemplateService struct {
	projectRepo  domain.ProjectRepository
	templateRepo domain.TemplateRepository
}

func NewTemplateService(projectRepo domain.ProjectRepository, templateRepo domain.TemplateRepository) *TemplateService {
	return &TemplateService{projectRepo: projectRepo, templateRepo: templateRepo}
}

func (s *TemplateService) CreateTemplate(ctx context.Context, projectID string, request dto.CreateTemplateRequest) (*domain.Template, error) {
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	request.Content = withUnsubscribeFooter(request.Content)
	if err := validateTemplate(request); err != nil {
		return nil, err
	}
	if request.Data == nil {
		request.Data = []string{}
	}
	record := &domain.Template{Name: request.Name, Subject: request.Subject, Data: request.Data, Content: request.Content, ProjectID: projectID, UserID: project.UserID}
	if err := s.templateRepo.Save(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TemplateService) FindAll(ctx context.Context, projectID string) ([]domain.Template, error) {
	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return nil, err
	}
	return s.templateRepo.FindByProjectID(ctx, projectID)
}

func (s *TemplateService) GetTemplate(ctx context.Context, projectID, name string) (*domain.Template, error) {
	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return nil, err
	}

	record, err := s.templateRepo.FindByNameAndProjectID(ctx, name, projectID)

	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *TemplateService) GetTemplateByID(ctx context.Context, projectID, id string) (*domain.Template, error) {
	if _, err := s.projectRepo.FindByID(ctx, projectID); err != nil {
		return nil, err
	}
	record, err := s.templateRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if record.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}
	return record, nil
}

func (s *TemplateService) UpdateTemplate(ctx context.Context, projectID, id string, request dto.CreateTemplateRequest) (*domain.Template, error) {
	record, err := s.GetTemplateByID(ctx, projectID, id)
	if err != nil {
		return nil, err
	}
	request.Content = withUnsubscribeFooter(request.Content)
	if err := validateTemplate(request); err != nil {
		return nil, err
	}
	if request.Data == nil {
		request.Data = []string{}
	}
	record.Name, record.Subject, record.Content, record.Data = request.Name, request.Subject, request.Content, request.Data
	if err := s.templateRepo.Update(ctx, record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *TemplateService) DeleteTemplate(ctx context.Context, projectID, id string) error {
	if _, err := s.GetTemplateByID(ctx, projectID, id); err != nil {
		return err
	}
	return s.templateRepo.Delete(ctx, id)
}

// unsubscribeFooter is appended to every stored template body, so no send can
// go out without a way for the reader to opt back out.
const unsubscribeFooter = `<p><a href="{{ .UnsubscribeURL }}">Unsubscribe</a></p>`

// unsubscribePlaceholder matches the action that renders a recipient's link,
// however the author spaced it.
var unsubscribePlaceholder = regexp.MustCompile(`\{\{-?\s*\.UnsubscribeURL\s*-?\}\}`)

// withUnsubscribeFooter appends the footer unless the body already places the
// link somewhere of its own, so an author can move it but not drop it. An empty
// body is left alone for validateTemplate to reject.
func withUnsubscribeFooter(content string) string {
	if strings.TrimSpace(content) == "" || unsubscribePlaceholder.MatchString(content) {
		return content
	}
	return strings.TrimRight(content, " \t\r\n") + "\n" + unsubscribeFooter
}

func validateTemplate(request dto.CreateTemplateRequest) error {
	if strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.Subject) == "" || strings.TrimSpace(request.Content) == "" || strings.ContainsAny(request.Subject, "\r\n") {
		return domain.ErrInvalidRequest
	}
	if _, err := template.New("body").Option("missingkey=error").Parse(request.Content); err != nil {
		return fmt.Errorf("%w: invalid HTML template", domain.ErrInvalidRequest)
	}
	if _, err := texttemplate.New("subject").Option("missingkey=error").Parse(request.Subject); err != nil {
		return fmt.Errorf("%w: invalid subject template", domain.ErrInvalidRequest)
	}
	return nil
}

func renderTemplate(record *domain.Template, data map[string]string) (string, string, error) {
	body, err := template.New("body").Option("missingkey=error").Parse(record.Content)
	if err != nil {
		return "", "", domain.ErrInvalidRequest
	}
	subject, err := texttemplate.New("subject").Option("missingkey=error").Parse(record.Subject)
	if err != nil {
		return "", "", domain.ErrInvalidRequest
	}
	var bodyBuffer, subjectBuffer bytes.Buffer
	if err := body.Execute(&bodyBuffer, data); err != nil {
		return "", "", fmt.Errorf("%w: template data is missing or invalid", domain.ErrInvalidRequest)
	}
	if err := subject.Execute(&subjectBuffer, data); err != nil {
		return "", "", fmt.Errorf("%w: subject data is missing or invalid", domain.ErrInvalidRequest)
	}
	renderedSubject := subjectBuffer.String()
	if strings.TrimSpace(renderedSubject) == "" || len(renderedSubject) > 998 || strings.ContainsAny(renderedSubject, "\r\n") {
		return "", "", domain.ErrInvalidRequest
	}
	return renderedSubject, bodyBuffer.String(), nil
}
