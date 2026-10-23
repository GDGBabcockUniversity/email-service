package services

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateKey_StoresHashAndReturnsSecretOnce(t *testing.T) {
	repo := new(mocks.ApiKeyRepository)
	projects := new(mocks.ProjectRepository)
	projects.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)
	repo.On("Save", mock.Anything, mock.Anything).Return(nil)
	record, secret, err := NewApiKeyService(repo, projects).CreateKey(context.Background(), "project")
	require.NoError(t, err)
	assert.Equal(t, hashKey(secret), record.Key)
	assert.NotEqual(t, secret, record.Key)
	encoded, err := json.Marshal(record)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), secret)
	assert.NotContains(t, string(encoded), record.Key)
}

func TestAuthenticate_MissingUnknownRevokedOrDeletedProject_ReturnsUnauthorized(t *testing.T) {
	for _, name := range []string{"missing", "unknown", "revoked", "deleted project"} {
		t.Run(name, func(t *testing.T) {
			repo := new(mocks.ApiKeyRepository)
			projects := new(mocks.ProjectRepository)
			secret := "secret"
			switch name {
			case "missing":
				secret = ""
			case "unknown":
				repo.On("FindByKey", mock.Anything, hashKey(secret)).Return(nil, domain.ErrNotFound)
			case "revoked":
				repo.On("FindByKey", mock.Anything, hashKey(secret)).Return(&domain.ApiKey{Revoked: true}, nil)
			default:
				repo.On("FindByKey", mock.Anything, hashKey(secret)).Return(&domain.ApiKey{ProjectID: "project"}, nil)
				projects.On("FindByID", mock.Anything, "project").Return(nil, domain.ErrNotFound)
			}
			_, err := NewApiKeyService(repo, projects).Authenticate(context.Background(), secret)
			assert.ErrorIs(t, err, domain.ErrUnauthorized)
		})
	}
}
