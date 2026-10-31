package services

import (
	"context"
	"encoding/json"
	sanity "github.com/sanity-io/client-go"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/stretchr/testify/assert"
)

func TestSanityImageURL(t *testing.T) {
	for name, testCase := range map[string]struct {
		projectID string
		dataset   string
		value     string
		want      string
	}{
		"asset reference becomes a cdn url": {
			projectID: "abc123", dataset: "production",
			value: "image-0a1b2c3d4e5f-1200x630-png",
			want:  "https://cdn.sanity.io/images/abc123/production/0a1b2c3d4e5f-1200x630.png",
		},
		"dereferenced url passes through": {
			projectID: "abc123", dataset: "production",
			value: "https://cdn.sanity.io/images/abc123/production/x-800x400.jpg",
			want:  "https://cdn.sanity.io/images/abc123/production/x-800x400.jpg",
		},
		"non production dataset": {
			projectID: "abc123", dataset: "staging",
			value: "image-0a1b-100x100-webp",
			want:  "https://cdn.sanity.io/images/abc123/staging/0a1b-100x100.webp",
		},
		"empty value": {
			projectID: "abc123", dataset: "production",
			value: "", want: "",
		},
		"reference without configured project": {
			dataset: "production",
			value:   "image-0a1b-100x100-png",
			want:    "",
		},
		"malformed reference": {
			projectID: "abc123", dataset: "production",
			value: "image-0a1b", want: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, testCase.want, sanityImageURL(testCase.projectID, testCase.dataset, testCase.value))
		})
	}
}

func TestSanityImageURL_FromAWebhookBody(t *testing.T) {
	for name, testCase := range map[string]struct {
		body string
		want string
	}{
		"plain projection leaves the asset as a reference": {
			body: `{"title":"Issue 4","coverImage":{"_type":"image","asset":{"_type":"reference","_ref":"image-0a1b2c-1200x630-png"}}}`,
			want: "https://cdn.sanity.io/images/abc123/production/0a1b2c-1200x630.png",
		},
		"projection dereferences the asset itself": {
			body: `{"title":"Issue 4","coverImage":"https://cdn.sanity.io/images/abc123/production/0a1b2c-1200x630.png"}`,
			want: "https://cdn.sanity.io/images/abc123/production/0a1b2c-1200x630.png",
		},
		"no cover on the document": {
			body: `{"title":"Issue 4"}`,
			want: "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var request dto.SanityWebhookRequest
			assert.NoError(t, json.Unmarshal([]byte(testCase.body), &request))
			assert.Equal(t, testCase.want, sanityImageURL("abc123", "production", request.CoverImage))
		})
	}
}

func TestSanityService_FetchPublished(t *testing.T) {
	for name, testCase := range map[string]struct {
		id, response, template string
		fields                 map[string]string
	}{
		"issue": {
			id: "abc", response: `{"Type":"issue","Title":"Issue 4","IssueNumber":"4","Slug":"issue-4","Cover":"c.png"}`, template: "newsletter",
			fields: map[string]string{"Title": "Issue 4", "Cover": "c.png", "IssueNumber": "4", "Slug": "issue-4"},
		},
		"article": {
			id: "abc", response: `{"Type":"article","Title":"Hello","Slug":"hello"}`, template: "article",
			fields: map[string]string{"Title": "Hello", "Cover": "", "IssueNumber": "", "Slug": "hello"},
		},
		"draft is skipped without a query": {id: "drafts.abc"},
		"missing document is skipped":      {id: "abc", response: `null`},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.NotEqual(t, "drafts.abc", testCase.id, "drafts must not hit Sanity")
				w.Write([]byte(`{"ms":1,"result":` + testCase.response + `}`))
			}))
			defer srv.Close()
			u, _ := url.Parse(srv.URL)
			s := NewSanityService("proj", "production", "secret", "", nil, sanity.WithBaseURL(*u))

			template, fields, err := s.fetchPublished(context.Background(), testCase.id)
			assert.NoError(t, err)
			assert.Equal(t, testCase.template, template)
			assert.Equal(t, testCase.fields, fields)
		})
	}
}
