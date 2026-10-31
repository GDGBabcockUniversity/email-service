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
	sanity "github.com/sanity-io/client-go"
)

type SanityService struct {
	ProjectID    string
	Dataset      string
	Secret       string
	EmailService *EmailService
	// Client is nil when the project or dataset is unset; the webhook then
	// trusts its payload instead of looking the document up.
	Client *sanity.Client
}

func NewSanityService(projectID, dataset, secret, token string, emailService *EmailService, opts ...sanity.Option) *SanityService {
	client, err := sanity.New(projectID, append([]sanity.Option{sanity.WithDataset(dataset), sanity.WithToken(token)}, opts...)...)
	if err != nil {
		slog.Warn("sanity client disabled", "err", err)
	}
	return &SanityService{
		Client:       client,
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

var sanityTemplates = map[string]string{"issue": "newsletter", "article": "article"}

func (s *SanityService) fetchPublished(ctx context.Context, id string) (string, map[string]string, error) {
	if strings.HasPrefix(id, "drafts.") {
		return "", nil, nil
	}

	result, err := s.Client.Query(`*[_id == $id && _type in ["issue", "article"]][0]{_type, title, "issueNumber": string(issueNumber), "slug": slug.current, "cover": coverImage.asset->url}`).
		Param("id", id).Do(ctx)
	if err != nil {
		return "", nil, err
	}
	var doc *struct{ Type, Title, IssueNumber, Slug, Cover string }
	if err := result.Unmarshal(&doc); err != nil {
		return "", nil, err
	}
	if doc == nil {
		return "", nil, nil
	}
	return sanityTemplates[doc.Type], map[string]string{"Title": doc.Title, "Cover": doc.Cover, "IssueNumber": doc.IssueNumber, "Slug": doc.Slug}, nil
}

func (s *SanityService) Webhook(ctx context.Context, projectID string, data dto.SanityWebhookRequest) error {

	template := sanityTemplates[data.Type]
	if template == "" {
		template = "newsletter"
	}
	fields := map[string]string{"Title": data.Title, "Cover": sanityImageURL(s.ProjectID, s.Dataset, data.CoverImage), "IssueNumber": data.IssueNumber, "Slug": data.Slug}
	if data.ID != "" && s.Client != nil {
		var err error
		if template, fields, err = s.fetchPublished(ctx, data.ID); err != nil {
			return err
		}
		if template == "" {
			slog.Info("Skipped unpublished or unsupported document", "id", data.ID)
			return nil
		}
	}

	_, err := s.EmailService.SendEmail(ctx, projectID, &dto.SendEmailRequest{
		To:       "all",
		Template: template,
		Data:     fields,
	})

	if err != nil {
		return err
	}

	slog.Info("Email sent")
	return nil

}
