package dto

import "github.com/GDGBabcockUnivsersity/email_service/internal/domain"

type SwaggerProjectResponse struct {
	Message string         `json:"message"`
	Data    domain.Project `json:"data"`
}

type SwaggerProjectsResponse struct {
	Message string           `json:"message"`
	Data    []domain.Project `json:"data"`
}

type SwaggerTemplateResponse struct {
	Message string          `json:"message"`
	Data    domain.Template `json:"data"`
}

type SwaggerTemplatesResponse struct {
	Message string            `json:"message"`
	Data    []domain.Template `json:"data"`
}

type SwaggerAPIKeyData struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Key       string `json:"key"`
	CreatedAt string `json:"created_at"`
}

type SwaggerApiKeySecretResponse struct {
	Message string            `json:"message"`
	Data    SwaggerAPIKeyData `json:"data"`
}

type SwaggerApiKeysResponse struct {
	Message string          `json:"message"`
	Data    []domain.ApiKey `json:"data"`
}

type SwaggerAcceptedEmailData struct {
	ID     string        `json:"id"`
	Status domain.Status `json:"status"`
}

type SwaggerAcceptedEmailResponse struct {
	Message string                   `json:"message"`
	Data    SwaggerAcceptedEmailData `json:"data"`
}
