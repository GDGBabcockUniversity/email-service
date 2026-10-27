package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
)

type SanityService struct {
	ProjectID    string
	Dataset      string
	Secret       string
	EmailService *EmailService
}

func NewSanityService(projectID, dataset, secret string, emailService *EmailService) *SanityService {
	return &SanityService{
		ProjectID:    projectID,
		Dataset:      dataset,
		Secret:       secret,
		EmailService: emailService,
	}
}

func sanityImageURL(projectID, dataset, value string) string {
	if !strings.HasPrefix(value, "image-") {
		return value
	}
	if projectID == "" || dataset == "" {
		return ""
	}
	parts := strings.Split(value, "-")
	if len(parts) < 4 {
		return ""
	}
	asset := strings.Join(parts[1:len(parts)-2], "-")
	dimensions := parts[len(parts)-2]
	extension := parts[len(parts)-1]
	if asset == "" || dimensions == "" || extension == "" {
		return ""
	}
	return fmt.Sprintf("https://cdn.sanity.io/images/%s/%s/%s-%s.%s", projectID, dataset, asset, dimensions, extension)
}

func (s *SanityService) VerifySignature(body []byte, signatureHeader, secret string) error {

	var timestamp, signature string

	for part := range strings.SplitSeq(signatureHeader, ",") {

		part = strings.TrimSpace(part)

		if after, found := strings.CutPrefix(part, "t="); found {
			timestamp = after
		}

		if after, found := strings.CutPrefix(part, "v1="); found {
			signature = after
		}
	}

	if timestamp == "" || signature == "" {
		return errors.New("Invalid signature")
	}

	var ts int64

	if _, err := fmt.Sscan(timestamp, &ts); err != nil {
		return errors.New("Invalid timestamp")
	}

	now := time.Now().UnixMilli()

	if ts < now-5000 || ts > now+5000 {
		return errors.New("Expired webhook")
	}

	received, err := base64.RawURLEncoding.DecodeString(signature)

	if err != nil {
		return err
	}

	messages := append([]byte(timestamp+"."), body...)

	mac := hmac.New(sha256.New, []byte(secret))

	mac.Write(messages)

	expected := mac.Sum(nil)

	if !hmac.Equal(received, expected) {
		return errors.New("Invalid signature")
	}

	return nil
}

func (s *SanityService) Webhook(ctx context.Context, projectID string, data dto.SanityWebhookRequest) error {

	_, err := s.EmailService.SendEmail(ctx, projectID, &dto.SendEmailRequest{
		To:       "all",
		Template: "newsletter",
		Data:     map[string]string{"Title": data.Title, "Cover": sanityImageURL(s.ProjectID, s.Dataset, data.CoverImage), "IssueNumber": data.IssueNumber, "Slug": data.Slug},
	})

	if err != nil {
		return err
	}

	slog.Info("Email sent")
	return nil

}
