package dto

import (
	"encoding/json"
	"strconv"
)

type CreateTemplateRequest struct {
	Name    string   `json:"name" binding:"required,max=200"`
	Subject string   `json:"subject" binding:"required,max=998"`
	Data    []string `json:"data"`
	Content string   `json:"content" binding:"required"`
}

type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
}

type SendEmailRequest struct {
	To       string            `json:"to" binding:"required,max=254"`
	Template string            `json:"template" binding:"required,max=200"`
	Data     map[string]string `json:"data"`
}

type AddAudienceRequest struct {
	Audiences []Audience `json:"audiences" binding:"required"`
}

type Audience struct {
	Email string `json:"email" binding:"required,email,max=254"`
}

type SanityWebhookRequest struct {
	Title       string `json:"title"`
	IssueNumber string `json:"issueNumber"`
	Slug        string `json:"slug"`
	CoverImage  string `json:"coverImage"`
}

var sanityFieldAliases = map[string][]string{
	"title":       {"title"},
	"issueNumber": {"issueNumber", "issue_number"},
	"slug":        {"slug"},
	"coverImage":  {"coverImage", "cover_image", "cover"},
}

func (r *SanityWebhookRequest) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	r.Title = sanityText(fields, "title")
	r.IssueNumber = sanityText(fields, "issueNumber")
	r.Slug = sanityText(fields, "slug")
	r.CoverImage = sanityText(fields, "coverImage")
	return nil
}

func sanityText(fields map[string]json.RawMessage, field string) string {
	for _, alias := range sanityFieldAliases[field] {
		raw, found := fields[alias]
		if !found {
			continue
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		if text := flattenSanityValue(value); text != "" {
			return text
		}
	}
	return ""
}

func flattenSanityValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case map[string]any:
		for _, key := range []string{"current", "url", "asset"} {
			if nested, found := typed[key]; found {
				if text := flattenSanityValue(nested); text != "" {
					return text
				}
			}
		}
	}
	return ""
}
