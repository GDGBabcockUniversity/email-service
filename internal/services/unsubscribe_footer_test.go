package services

import (
	"testing"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The footer the service appends has to survive template validation and then
// render into a real link for a recipient.
func TestUnsubscribeFooter_ValidatesAndRenders(t *testing.T) {
	content := withUnsubscribeFooter("<h1>Hello {{.name}}</h1>")

	require.NoError(t, validateTemplate(dto.CreateTemplateRequest{Name: "news", Subject: "News", Content: content}))

	tokens := NewUnsubscribeTokens("unsub-secret", "https://mail.example.com")
	service := &EmailService{tokens: tokens}
	_, body, err := renderTemplate(
		&domain.Template{Subject: "News", Content: content},
		service.recipientData(map[string]string{"name": "Alice"}, "project", "alice@example.com"),
	)

	require.NoError(t, err)
	assert.Contains(t, body, "href=\""+tokens.URL("project", "alice@example.com")+"\"")
	assert.Contains(t, body, ">Unsubscribe</a>")
}
