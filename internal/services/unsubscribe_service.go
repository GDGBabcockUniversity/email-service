package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
)

type UnsubscribeTokens struct {
	secret  []byte
	baseURL string
}

func NewUnsubscribeTokens(secret, baseURL string) *UnsubscribeTokens {
	return &UnsubscribeTokens{secret: []byte(secret), baseURL: strings.TrimRight(baseURL, "/")}
}

// Token returns the raw token for a subscription. It is safe to put in a link.
func (t *UnsubscribeTokens) Token(projectID, email string) string {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(projectID + "|" + normalizeEmail(email)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Hash returns the value stored in the database for a raw token.
func (t *UnsubscribeTokens) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// HashFor is the stored hash of a subscription's token.
func (t *UnsubscribeTokens) HashFor(projectID, email string) string {
	return t.Hash(t.Token(projectID, email))
}

// unsubscribeLinkPath is the reader-facing confirmation page, not this
// service's own endpoint: a recipient clicking from their inbox should land on
// a page that asks before opting them out. That page posts the token back to
// POST /api/unsubscribe, which stays available for direct use.
const unsubscribeLinkPath = "/unsubscribe"

// URL is the unsubscribe link to embed in a recipient's email. It points at
// PublicBaseURL, so that must be the origin serving the confirmation page.
//
// The token rides in the path rather than the query string because an ESP's
// click tracking rewrites the link and can drop the query string with it. Brevo
// does exactly that, which leaves the reader on the page with nothing to
// identify their subscription. A path segment stands a better chance of
// surviving the round trip.
func (t *UnsubscribeTokens) URL(projectID, email string) string {
	return t.baseURL + unsubscribeLinkPath + "/" + url.PathEscape(t.Token(projectID, email))
}

// UnsubscribeService resolves a token from an unsubscribe link back to the
// subscription that owns it and records the recipient's choice.
type UnsubscribeService struct {
	audienceRepo domain.AudienceRepository
	tokens       *UnsubscribeTokens
}

func NewUnsubscribeService(audienceRepo domain.AudienceRepository, tokens *UnsubscribeTokens) *UnsubscribeService {
	return &UnsubscribeService{audienceRepo: audienceRepo, tokens: tokens}
}

// Unsubscribe marks the token's subscription as no longer subscribed. It is
// idempotent: a token used twice reports the same already-unsubscribed row.
func (s *UnsubscribeService) Unsubscribe(ctx context.Context, token string) (*domain.Audience, error) {
	return s.setSubscribed(ctx, token, false)
}

// Resubscribe turns the same subscription back on, reusing the existing row so
// a project never collects a second audience entry for one address.
func (s *UnsubscribeService) Resubscribe(ctx context.Context, token string) (*domain.Audience, error) {
	return s.setSubscribed(ctx, token, true)
}

func (s *UnsubscribeService) setSubscribed(ctx context.Context, token string, subscribed bool) (*domain.Audience, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, domain.ErrInvalidRequest
	}
	audience, err := s.audienceRepo.FindByUnsubscribeToken(ctx, s.tokens.Hash(token))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if audience == nil {
		return nil, domain.ErrNotFound
	}
	if audience.Subscribed == subscribed {
		return audience, nil
	}
	if err := s.audienceRepo.SetSubscribed(ctx, audience.ID, subscribed); err != nil {
		return nil, err
	}
	audience.Subscribed = subscribed
	return audience, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
