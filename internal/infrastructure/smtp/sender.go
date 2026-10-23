package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	netsmtp "net/smtp"
	"strings"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/config"
)

type Sender struct {
	config *config.Config
}

func NewSender(cfg *config.Config) *Sender {
	return &Sender{config: cfg}
}

func (s *Sender) Send(ctx context.Context, email *domain.Email) error {
	if strings.ContainsAny(email.To+email.Subject, "\r\n") {
		return fmt.Errorf("invalid email headers")
	}
	recipient, err := mail.ParseAddress(email.To)
	if err != nil || recipient.Address != email.To {
		return fmt.Errorf("invalid recipient")
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	address := net.JoinHostPort(s.config.SMTPHost, s.config.SMTPPort)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer connection.Close()
	stop := context.AfterFunc(ctx, func() { connection.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()

	if err := connection.SetDeadline(deadline); err != nil {
		return err
	}

	tlsConfig := &tls.Config{ServerName: s.config.SMTPHost, MinVersion: tls.VersionTLS12}

	var smtpConnection net.Conn = connection

	if s.config.SMTPTLSMode == "tls" {
		secure := tls.Client(connection, tlsConfig)
		if err := secure.HandshakeContext(ctx); err != nil {
			return err
		}
		smtpConnection = secure
	}
	client, err := netsmtp.NewClient(smtpConnection, s.config.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()
	if s.config.SMTPTLSMode == "starttls" {
		if err := client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if s.config.SMTPUsername != "" {
		auth := netsmtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, s.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(s.config.SMTPFrom); err != nil {
		return err
	}
	if err := client.Rcpt(email.To); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: <%s@gdg-email.local>\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n",
		s.config.SMTPFrom, email.To, mime.QEncoding.Encode("utf-8", email.Subject), time.Now().UTC().Format(time.RFC1123Z), email.ID)
	if _, err := writer.Write([]byte(headers)); err != nil {
		return err
	}
	encoder := quotedprintable.NewWriter(writer)
	if _, err := encoder.Write([]byte(email.Body)); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	// The DATA response confirms acceptance; QUIT failure must not trigger another send.
	_ = client.Quit()
	return nil
}
