package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/mocks"
	"github.com/GDGBabcockUnivsersity/email_service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestManagement_MissingOrWrongAdminKey_ReturnsUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter("admin-secret", nil, nil, nil, nil, nil, nil)
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/projects"},
		{http.MethodGet, "/api/projects/p/audience"},
		{http.MethodDelete, "/api/projects/p/audience/a"},
		{http.MethodGet, "/api/projects/p/emails"},
		{http.MethodGet, "/api/projects/p/emails/stats"},
		{http.MethodPost, "/api/projects/p/emails"},
	}
	for _, route := range routes {
		for _, key := range []string{"", "wrong"} {
			request := httptest.NewRequest(route.method, route.path, nil)
			request.Header.Set("X-Admin-Key", key)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			assert.Equal(t, http.StatusUnauthorized, response.Code, route.path)
			assert.JSONEq(t, `{"message":"Unauthorized","data":null}`, response.Body.String())
		}
	}
}

func TestSendEmail_InvalidJSON_ReturnsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	apiKeyRepo := new(mocks.ApiKeyRepository)
	projectRepo := new(mocks.ProjectRepository)
	emailRepo := new(mocks.EmailRepository)

	apiKeyRepo.On("FindByKey", mock.Anything, mock.Anything).Return(&domain.ApiKey{ProjectID: "project"}, nil)
	projectRepo.On("FindByID", mock.Anything, "project").Return(&domain.Project{}, nil)
	router := NewRouter("admin", nil, nil, services.NewApiKeyService(apiKeyRepo, projectRepo), nil, services.NewSanityService("proj", "production", "secret", "", services.NewEmailService(emailRepo, nil, nil, nil, services.NewUnsubscribeTokens("unsub-secret", "http://localhost:8080"))), nil)

	for _, body := range []string{"{", `{"to":"not-an-email","template":"bad"}`, `{}`} {
		request := httptest.NewRequest(http.MethodPost, "/api/email", strings.NewReader(body))
		request.Header.Set("x-gdg_email-apikey", "secret")
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		assert.Equal(t, http.StatusBadRequest, response.Code)
	}
}

func TestAdminDashboard_ServesBuiltConsoleWithoutAdminKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dist := t.TempDir()
	assert.NoError(t, os.MkdirAll(filepath.Join(dist, "assets"), 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(dist, "index.html"), []byte("<div id=root></div>"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(dist, "assets", "app.js"), []byte("console.log(1)"), 0o644))
	t.Setenv("WEB_DIST", dist)
	router := NewRouter("admin-secret", nil, nil, nil, nil, nil, nil)
	for path, want := range map[string]struct {
		code int
		body string
	}{
		"/admin":               {http.StatusMovedPermanently, ""},
		"/admin/":              {http.StatusOK, "<div id=root>"},
		"/admin/assets":        {http.StatusOK, "<div id=root>"},
		"/admin/assets/app.js": {http.StatusOK, "console.log(1)"},
		"/admin/anything":      {http.StatusOK, "<div id=root>"},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, want.code, response.Code, path)
		assert.Contains(t, response.Body.String(), want.body, path)
	}
}

// Recipients click the unsubscribe link straight from their inbox, so the route
// must work with no admin key, and the token must select the subscription.
func TestUnsubscribe_PublicRouteResolvesTheTokenWithoutAnAdminKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	audienceRepo := new(mocks.AudienceRepository)
	tokens := services.NewUnsubscribeTokens("unsub-secret", "http://localhost:8080")
	token := tokens.Token("project", "alice@example.com")

	audienceRepo.On("FindByUnsubscribeToken", mock.Anything, tokens.Hash(token)).
		Return(&domain.Audience{ID: "a1", Email: "alice@example.com", ProjectID: "project", Subscribed: true}, nil).Once()
	audienceRepo.On("SetSubscribed", mock.Anything, "a1", false).Return(nil).Once()

	router := NewRouter("admin-secret", nil, nil, nil, nil, nil, services.NewUnsubscribeService(audienceRepo, tokens))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/unsubscribe?token="+token, nil))

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"message":"Unsubscribed","data":{"email":"alice@example.com","project_id":"project","subscribed":false}}`, response.Body.String())
	audienceRepo.AssertExpectations(t)
}

func TestUnsubscribe_MissingOrUnknownToken_ReturnsBadRequestOrNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	audienceRepo := new(mocks.AudienceRepository)
	tokens := services.NewUnsubscribeTokens("unsub-secret", "http://localhost:8080")
	audienceRepo.On("FindByUnsubscribeToken", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound)
	router := NewRouter("admin-secret", nil, nil, nil, nil, nil, services.NewUnsubscribeService(audienceRepo, tokens))

	for path, expected := range map[string]int{
		"/api/unsubscribe":             http.StatusBadRequest,
		"/api/unsubscribe?token=bogus": http.StatusNotFound,
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		assert.Equal(t, expected, response.Code, path)
	}
}
