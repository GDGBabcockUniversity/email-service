package dto

type CreateTemplateRequest struct {
	Name    string   `json:"name" binding:"required,max=200"`
	Subject string   `json:"subject" binding:"required,max=998"`
	Data    []string `json:"data"`
	Content string   `json:"content" binding:"required"`
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
}

type SendEmailRequest struct {
	To       string            `json:"to" binding:"required,max=254"`
	Template string            `json:"template" binding:"required,max=200"`
	Data     map[string]string `json:"data"`
}

type AddAudienceRequest struct {
	Audiences []Audience `json:"audiences" binding:"required"`
}

type Audience struct {
	Email string `json:"email" binding:"required,email,max=254"`
}

type SanityWebhookRequest struct {
	Title string `json:"title"`
}
