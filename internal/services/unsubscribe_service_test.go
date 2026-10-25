package services

import (
	"context"
	"testing"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testTokens() *UnsubscribeTokens {
	return NewUnsubscribeTokens("unsub-secret", "https://mail.example.com/")
}

// The database must never hold the token that travels in the email link.
func TestUnsubscribeTokens_StoresOnlyTheHashAndBuildsALink(t *testing.T) {
	tokens := testTokens()

	token := tokens.Token("project", "Alice@Example.com")

	assert.NotEqual(t, token, tokens.HashFor("project", "alice@example.com"))
	assert.Equal(t, tokens.Hash(token), tokens.HashFor("project", "alice@example.com"))
	// Address casing and surrounding space describe one subscription.
	assert.Equal(t, token, tokens.Token("project", " alice@example.com "))
	// A different project or secret yields a different token.
	assert.NotEqual(t, token, tokens.Token("other-project", "alice@example.com"))
	assert.NotEqual(t, token, NewUnsubscribeTokens("other-secret", "https://mail.example.com").Token("project", "alice@example.com"))
	assert.Equal(t, "https://mail.example.com/api/unsubscribe?token="+token, tokens.URL("project", "alice@example.com"))
}

func TestUnsubscribe_HashesTheTokenAndFlipsTheSubscription(t *testing.T) {
	audienceRepo := new(mocks.AudienceRepository)
	tokens := testTokens()
	token := tokens.Token("project", "alice@example.com")

	audienceRepo.On("FindByUnsubscribeToken", mock.Anything, tokens.Hash(token)).
		Return(&domain.Audience{ID: "audience", Email: "alice@example.com", ProjectID: "project", Subscribed: true}, nil).Once()
	audienceRepo.On("SetSubscribed", mock.Anything, "audience", false).Return(nil).Once()

	service := NewUnsubscribeService(audienceRepo, tokens)
	audience, err := service.Unsubscribe(context.Background(), token)

	require.NoError(t, err)
	assert.False(t, audience.Subscribed)
	audienceRepo.AssertExpectations(t)
}

// Mail clients and recipients re-open links; a second click must not error.
func TestUnsubscribe_AlreadyUnsubscribed_IsIdempotent(t *testing.T) {
	audienceRepo := new(mocks.AudienceRepository)
	tokens := testTokens()
	token := tokens.Token("project", "alice@example.com")

	audienceRepo.On("FindByUnsubscribeToken", mock.Anything, tokens.Hash(token)).
		Return(&domain.Audience{ID: "audience", ProjectID: "project", Subscribed: false}, nil).Once()

	audience, err := NewUnsubscribeService(audienceRepo, tokens).Unsubscribe(context.Background(), token)

	require.NoError(t, err)
	assert.False(t, audience.Subscribed)
	audienceRepo.AssertNotCalled(t, "SetSubscribed", mock.Anything, mock.Anything, mock.Anything)
}

func TestUnsubscribe_UnknownOrEmptyToken_ReturnsNotFoundOrBadRequest(t *testing.T) {
	audienceRepo := new(mocks.AudienceRepository)
	tokens := testTokens()
	audienceRepo.On("FindByUnsubscribeToken", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound)

	service := NewUnsubscribeService(audienceRepo, tokens)

	_, err := service.Unsubscribe(context.Background(), "   ")
	assert.ErrorIs(t, err, domain.ErrInvalidRequest)

	_, err = service.Unsubscribe(context.Background(), "not-a-real-token")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestResubscribe_TurnsTheSameSubscriptionBackOn(t *testing.T) {
	audienceRepo := new(mocks.AudienceRepository)
	tokens := testTokens()
	token := tokens.Token("project", "alice@example.com")

	audienceRepo.On("FindByUnsubscribeToken", mock.Anything, tokens.Hash(token)).
		Return(&domain.Audience{ID: "audience", ProjectID: "project", Subscribed: false}, nil).Once()
	audienceRepo.On("SetSubscribed", mock.Anything, "audience", true).Return(nil).Once()

	audience, err := NewUnsubscribeService(audienceRepo, tokens).Resubscribe(context.Background(), token)

	require.NoError(t, err)
	assert.True(t, audience.Subscribed)
	audienceRepo.AssertExpectations(t)
}

// Re-adding an address that unsubscribed must revive its row, not create a
// second audience entry for the project.
func TestAddAudience_ExistingAddress_ReusesTheRowInsteadOfAddingAnother(t *testing.T) {
	projectRepo := new(mocks.ProjectRepository)
	audienceRepo := new(mocks.AudienceRepository)
	tokens := testTokens()

	audienceRepo.On("FindByEmailAndProjectID", mock.Anything, "alice@example.com", "project").
		Return(&domain.Audience{ID: "audience", Email: "alice@example.com", ProjectID: "project", Subscribed: false}, nil).Once()
	audienceRepo.On("FindByEmailAndProjectID", mock.Anything, "bob@example.com", "project").
		Return(nil, domain.ErrNotFound).Once()
	audienceRepo.On("SetSubscribed", mock.Anything, "audience", true).Return(nil).Once()
	audienceRepo.On("SaveAll", mock.Anything, []domain.Audience{{
		Email:            "bob@example.com",
		ProjectID:        "project",
		Subscribed:       true,
		UnsubscribeToken: tokens.HashFor("project", "bob@example.com"),
	}}).Return(nil).Once()

	service := NewProjectService(projectRepo, audienceRepo, tokens, "owner")
	// "Bob" repeats to prove one request cannot insert the same address twice.
	err := service.AddAudience(context.Background(), "project", dto.AddAudienceRequest{Audiences: []dto.Audience{
		{Email: "Alice@Example.com"}, {Email: "bob@example.com"}, {Email: "BOB@example.com"},
	}})

	require.NoError(t, err)
	audienceRepo.AssertExpectations(t)
}
