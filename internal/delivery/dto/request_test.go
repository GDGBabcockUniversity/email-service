package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanityWebhookRequest_DecodesEveryProjectionShape(t *testing.T) {
	for name, testCase := range map[string]struct {
		body string
		want SanityWebhookRequest
	}{
		"flattened by the projection": {
			body: `{"title":"Issue 4","issueNumber":"4","slug":"issue-4","coverImage":"https://cdn.sanity.io/images/x.png"}`,
			want: SanityWebhookRequest{Title: "Issue 4", IssueNumber: "4", Slug: "issue-4", CoverImage: "https://cdn.sanity.io/images/x.png"},
		},
		"snake_case keys": {
			body: `{"title":"Issue 4","issue_number":"4","slug":"issue-4","cover_image":"https://cdn.sanity.io/images/x.png"}`,
			want: SanityWebhookRequest{Title: "Issue 4", IssueNumber: "4", Slug: "issue-4", CoverImage: "https://cdn.sanity.io/images/x.png"},
		},
		"raw document, nothing flattened": {
			body: `{"_id":"abc","_type":"issue","title":"Issue 4","issueNumber":4,"slug":{"_type":"slug","current":"issue-4"},"coverImage":{"_type":"image","asset":{"url":"https://cdn.sanity.io/images/x.png"}}}`,
			want: SanityWebhookRequest{Title: "Issue 4", IssueNumber: "4", Slug: "issue-4", CoverImage: "https://cdn.sanity.io/images/x.png"},
		},
		"asset left as a reference carries no url": {
			body: `{"title":"Issue 4","coverImage":{"asset":{"_ref":"image-xyz-1200x630-png"}}}`,
			want: SanityWebhookRequest{Title: "Issue 4"},
		},
		"deleted document projects nulls": {
			body: `{"title":null,"issueNumber":null,"slug":null,"coverImage":null}`,
			want: SanityWebhookRequest{},
		},
		"fields absent entirely": {
			body: `{}`,
			want: SanityWebhookRequest{},
		},
	} {
		t.Run(name, func(t *testing.T) {
			var request SanityWebhookRequest
			assert.NoError(t, json.Unmarshal([]byte(testCase.body), &request))
			assert.Equal(t, testCase.want, request)
		})
	}
}

func TestSanityWebhookRequest_RejectsANonObjectBody(t *testing.T) {
	var request SanityWebhookRequest
	assert.Error(t, json.Unmarshal([]byte(`["issue-4"]`), &request))
}
