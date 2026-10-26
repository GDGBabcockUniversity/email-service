package smtp

import (
	"context"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/config"
	"github.com/GDGBabcockUnivsersity/email_service/internal/mocks"
	"github.com/GDGBabcockUnivsersity/email_service/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSend_LocalSMTP_ProducesValidHTMLMessage(t *testing.T) {
	host, port, messages := mocks.NewSMTPServer(t)
	sender := NewSender(&config.Config{SMTPHost: host, SMTPPort: port, SMTPFrom: "gdg@example.com", SMTPTLSMode: "none"})
	email := &domain.Email{ID: "test-email", To: "alice@example.com", Subject: "Hello 👋", Body: "<p>Hello Alice = welcome</p>"}
	require.NoError(t, sender.Send(context.Background(), email))
	select {
	case raw := <-messages:
		message, err := mail.ReadMessage(strings.NewReader(string(raw)))
		require.NoError(t, err)
		subject, err := new(mime.WordDecoder).DecodeHeader(message.Header.Get("Subject"))
		require.NoError(t, err)
		assert.Equal(t, email.Subject, subject)
		decoded, err := io.ReadAll(quotedprintable.NewReader(message.Body))
		require.NoError(t, err)
		assert.Equal(t, email.Body, strings.TrimSuffix(string(decoded), "\n"))
		assert.Equal(t, email.To, message.Header.Get("To"))
	case <-time.After(time.Second):
		t.Fatal("SMTP message not received")
	}
}

func TestSend_STARTTLSRequired_RejectsPlaintextServer(t *testing.T) {
	host, port, _ := mocks.NewSMTPServer(t)
	sender := NewSender(&config.Config{SMTPHost: host, SMTPPort: port, SMTPFrom: "gdg@example.com", SMTPTLSMode: "starttls"})
	assert.Error(t, sender.Send(context.Background(), &domain.Email{To: "alice@example.com", Subject: "Hello"}))
}

// The header is the only copy of the token that survives an ESP rewriting every
// link in the body, so it has to reach the wire verbatim.
func TestSend_EmitsListUnsubscribeHeaders(t *testing.T) {
	host, port, messages := mocks.NewSMTPServer(t)
	cfg := &config.Config{
		SMTPHost: host, SMTPPort: port, SMTPFrom: "gdg@example.com", SMTPTLSMode: "none",
		UnsubscribeSecret: "unsub-secret", PublicBaseURL: "https://api.example.com",
	}
	sender := NewSender(cfg)
	email := &domain.Email{ID: "test-email", To: "alice@example.com", ProjectID: "project", Subject: "Hello", Body: "<p>Hi</p>"}

	require.NoError(t, sender.Send(context.Background(), email))

	select {
	case raw := <-messages:
		message, err := mail.ReadMessage(strings.NewReader(string(raw)))
		require.NoError(t, err)
		token := services.NewUnsubscribeTokens("unsub-secret", "https://api.example.com").Token("project", "alice@example.com")
		assert.Equal(t, "<https://api.example.com/api/unsubscribe?token="+token+">", message.Header.Get("List-Unsubscribe"))
		assert.Equal(t, "List-Unsubscribe=One-Click", message.Header.Get("List-Unsubscribe-Post"))
		// Message-ID must still carry the email's own ID, and the body survive.
		assert.Equal(t, "<test-email@gdg-email.local>", message.Header.Get("Message-ID"))
		decoded, err := io.ReadAll(quotedprintable.NewReader(message.Body))
		require.NoError(t, err)
		assert.Equal(t, "<p>Hi</p>", strings.TrimSuffix(string(decoded), "\n"))
	case <-time.After(time.Second):
		t.Fatal("SMTP message not received")
	}
}

// A one-off send with no project has no subscription to point at, so no
// half-built header should ship.
func TestSend_NoProject_OmitsListUnsubscribeHeaders(t *testing.T) {
	host, port, messages := mocks.NewSMTPServer(t)
	sender := NewSender(&config.Config{
		SMTPHost: host, SMTPPort: port, SMTPFrom: "gdg@example.com", SMTPTLSMode: "none",
		UnsubscribeSecret: "unsub-secret", PublicBaseURL: "https://api.example.com",
	})

	require.NoError(t, sender.Send(context.Background(), &domain.Email{ID: "test-email", To: "alice@example.com", Subject: "Hello", Body: "<p>Hi</p>"}))

	select {
	case raw := <-messages:
		message, err := mail.ReadMessage(strings.NewReader(string(raw)))
		require.NoError(t, err)
		assert.Empty(t, message.Header.Get("List-Unsubscribe"))
		assert.Empty(t, message.Header.Get("List-Unsubscribe-Post"))
	case <-time.After(time.Second):
		t.Fatal("SMTP message not received")
	}
}

// A base URL carrying a newline would let the header block inject arbitrary
// headers; it is dropped rather than written to the wire.
func TestListUnsubscribeHeaders_MalformedBaseURL_IsDropped(t *testing.T) {
	for name, base := range map[string]string{
		"header split": "https://api.example.com/\r\nBcc: eve@example.com",
		"not absolute": "/api",
		"empty":        "",
	} {
		t.Run(name, func(t *testing.T) {
			sender := NewSender(&config.Config{UnsubscribeSecret: "unsub-secret", PublicBaseURL: base})
			assert.Empty(t, sender.listUnsubscribeHeaders(&domain.Email{To: "alice@example.com", ProjectID: "project"}))
		})
	}
}
