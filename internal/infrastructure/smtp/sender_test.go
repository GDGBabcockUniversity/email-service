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
