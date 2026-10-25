package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/services"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	projectService  *services.ProjectService
	templateService *services.TemplateService
	apiKeyService   *services.ApiKeyService
	emailService    *services.EmailService
	sanityService   *services.SanityService
}

// NewHandler creates an HTTP handler with the supplied services.
func NewHandler(projectService *services.ProjectService, templateService *services.TemplateService, apiKeyService *services.ApiKeyService, emailService *services.EmailService, sanityService *services.SanityService) *Handler {
	return &Handler{
		projectService:  projectService,
		templateService: templateService,
		apiKeyService:   apiKeyService,
		emailService:    emailService,
		sanityService:   sanityService,
	}
}

// sendEmail godoc
// @Summary Queue an email
// @Description Renders a project template with the supplied data and queues the email for SMTP delivery.
// @Tags Email
// @Security ProjectApiKey
// @Accept json
// @Produce json
// @Param request body dto.SendEmailRequest true "Email recipient and template data"
// @Success 202 {object} dto.SwaggerAcceptedEmailResponse
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/email [post]
func (h *Handler) sendEmail(c *gin.Context) {
	var request dto.SendEmailRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, domain.ErrInvalidRequest)
		return
	}
	emails, err := h.emailService.SendEmail(c.Request.Context(), c.GetString("projectID"), &request)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, dto.ApiResponse[any]{Message: "Email accepted", Data: queued(emails)})
}

// queued summarises the rows a send created, in the order they were accepted.
func queued(emails []*domain.Email) []map[string]string {
	summary := []map[string]string{}
	for _, email := range emails {
		summary = append(summary, map[string]string{"id": email.ID, "status": string(email.Status)})
	}
	return summary
}

// createProject godoc
// @Summary Create a project
// @Tags Projects
// @Security AdminKey
// @Accept json
// @Produce json
// @Param request body dto.CreateProjectRequest true "Project name and description"
// @Success 201 {object} dto.SwaggerProjectResponse
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects [post]
func (h *Handler) createProject(c *gin.Context) {
	var request dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, domain.ErrInvalidRequest)
		return
	}
	project, err := h.projectService.CreateProject(c.Request.Context(), &request)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.ApiResponse[any]{Message: "Project created", Data: project})
}

// listProjects godoc
// @Summary List projects
// @Tags Projects
// @Security AdminKey
// @Produce json
// @Success 200 {object} dto.SwaggerProjectsResponse
// @Failure 401 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects [get]
func (h *Handler) listProjects(c *gin.Context) {
	projects, err := h.projectService.FindAll(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Projects retrieved", Data: projects})
}

// getProject godoc
// @Summary Get a project
// @Tags Projects
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Success 200 {object} dto.SwaggerProjectResponse
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId} [get]
func (h *Handler) getProject(c *gin.Context) {
	project, err := h.projectService.FindByID(c.Request.Context(), c.Param("projectId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Project retrieved", Data: project})
}

// updateProject godoc
// @Summary Update a project
// @Description Replaces the project's name and description.
// @Tags Projects
// @Security AdminKey
// @Accept json
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param request body dto.CreateProjectRequest true "Updated project"
// @Success 200 {object} dto.SwaggerProjectResponse
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId} [put]
func (h *Handler) updateProject(c *gin.Context) {
	var request dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, domain.ErrInvalidRequest)
		return
	}
	project, err := h.projectService.UpdateProject(c.Request.Context(), c.Param("projectId"), &request)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Project updated", Data: project})
}

// deleteProject godoc
// @Summary Soft-delete a project
// @Description Also soft-deletes its templates and revokes its API keys.
// @Tags Projects
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Success 200 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId} [delete]
func (h *Handler) deleteProject(c *gin.Context) {
	if err := h.projectService.DeleteProject(c.Request.Context(), c.Param("projectId")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Project deleted", Data: nil})
}

// createTemplate godoc
// @Summary Create a project template
// @Tags Templates
// @Security AdminKey
// @Accept json
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param request body dto.CreateTemplateRequest true "Template subject, content, and example data"
// @Success 201 {object} dto.SwaggerTemplateResponse
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/templates [post]
func (h *Handler) createTemplate(c *gin.Context) {
	var request dto.CreateTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, domain.ErrInvalidRequest)
		return
	}
	template, err := h.templateService.CreateTemplate(c.Request.Context(), c.Param("projectId"), request)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.ApiResponse[any]{Message: "Template created", Data: template})
}

// listTemplates godoc
// @Summary List a project's templates
// @Tags Templates
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Success 200 {object} dto.SwaggerTemplatesResponse
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/templates [get]
func (h *Handler) listTemplates(c *gin.Context) {
	templates, err := h.templateService.FindAll(c.Request.Context(), c.Param("projectId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Templates retrieved", Data: templates})
}

// getTemplate godoc
// @Summary Get a template
// @Tags Templates
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param id path string true "Template UUID"
// @Success 200 {object} dto.SwaggerTemplateResponse
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/templates/{id} [get]
func (h *Handler) getTemplate(c *gin.Context) {
	template, err := h.templateService.GetTemplateByID(c.Request.Context(), c.Param("projectId"), c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Template retrieved", Data: template})
}

// updateTemplate godoc
// @Summary Update a template
// @Description Replaces the template's name, subject, content, and example data.
// @Tags Templates
// @Security AdminKey
// @Accept json
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param id path string true "Template UUID"
// @Param request body dto.CreateTemplateRequest true "Updated template"
// @Success 200 {object} dto.SwaggerTemplateResponse
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 409 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/templates/{id} [put]
func (h *Handler) updateTemplate(c *gin.Context) {
	var request dto.CreateTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, domain.ErrInvalidRequest)
		return
	}
	template, err := h.templateService.UpdateTemplate(c.Request.Context(), c.Param("projectId"), c.Param("id"), request)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Template updated", Data: template})
}

// deleteTemplate godoc
// @Summary Soft-delete a template
// @Tags Templates
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param id path string true "Template UUID"
// @Success 200 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/templates/{id} [delete]
func (h *Handler) deleteTemplate(c *gin.Context) {
	if err := h.templateService.DeleteTemplate(c.Request.Context(), c.Param("projectId"), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Template deleted", Data: nil})
}

// createKey godoc
// @Summary Create a project API key
// @Description The raw key is returned once; store it securely.
// @Tags API keys
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Success 201 {object} dto.SwaggerApiKeySecretResponse
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/api-keys [post]
func (h *Handler) createKey(c *gin.Context) {
	key, rawKey, err := h.apiKeyService.CreateKey(c.Request.Context(), c.Param("projectId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, dto.ApiResponse[any]{Message: "API key created", Data: gin.H{"id": key.ID, "project_id": key.ProjectID, "key": rawKey, "created_at": key.CreatedAt}})
}

// listKeys godoc
// @Summary List API key metadata for a project
// @Description Does not return raw keys or stored hashes.
// @Tags API keys
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Success 200 {object} dto.SwaggerApiKeysResponse
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/api-keys [get]
func (h *Handler) listKeys(c *gin.Context) {
	keys, err := h.apiKeyService.FindAll(c.Request.Context(), c.Param("projectId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "API keys retrieved", Data: keys})
}

// revokeKey godoc
// @Summary Revoke a project API key
// @Tags API keys
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param id path string true "API key UUID"
// @Success 200 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/api-keys/{id} [delete]
func (h *Handler) revokeKey(c *gin.Context) {
	if err := h.apiKeyService.RevokeKey(c.Request.Context(), c.Param("projectId"), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "API key revoked", Data: nil})
}

// addAudience godoc
// @Summary Add audience to a project
// @Tags Audience
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param audience body dto.AddAudienceRequest true "Audience to add"
// @Success 200 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/audience [post]
func (h *Handler) addAudience(c *gin.Context) {

	var audience dto.AddAudienceRequest

	if err := c.ShouldBindJSON(&audience); err != nil {
		respondError(c, err)
		return
	}

	if err := h.projectService.AddAudience(c.Request.Context(), c.Param("projectId"), audience); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Audience added", Data: nil})
}

func (h *Handler) sanityWebhook(c *gin.Context) {

	body, err := io.ReadAll(c.Request.Body)

	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid body"})
		return
	}

	signature := c.GetHeader("sanity-webhook-signature")

	projectID := c.GetHeader("project-id")

	slog.Info("id", "project id", projectID)

	if err := h.sanityService.VerifySignature(body, signature, h.sanityService.Secret); err != nil {
		slog.Error("Error", "err", err)
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	var sanityWebhookRequest dto.SanityWebhookRequest

	if err := json.Unmarshal(body, &sanityWebhookRequest); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
		return
	}

	_ = h.sanityService.Webhook(c.Request.Context(), projectID, sanityWebhookRequest)
	c.JSON(http.StatusOK, map[string]string{"status": "done"})
}

// sendProjectEmail godoc
// @Summary Send one of a project's templates
// @Description Renders the named template and queues it. Set "to" to "all" to reach every address in the project's audience, or to a single email address.
// @Tags Email
// @Security AdminKey
// @Accept json
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param request body dto.SendEmailRequest true "Recipient, template name, and template data"
// @Success 202 {object} dto.SwaggerAcceptedEmailResponse
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/emails [post]
func (h *Handler) sendProjectEmail(c *gin.Context) {
	var request dto.SendEmailRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		respondError(c, domain.ErrInvalidRequest)
		return
	}
	emails, err := h.emailService.SendEmail(c.Request.Context(), c.Param("projectId"), &request)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, dto.ApiResponse[any]{Message: "Email accepted", Data: queued(emails)})
}

// listEmails godoc
// @Summary List a project's emails
// @Description Returns the most recent emails for the project, newest first.
// @Tags Email
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param status query string false "Filter by delivery status" Enums(PENDING, PROCESSING, SENT, FAILED)
// @Param limit query int false "Maximum emails to return (1-200, default 50)"
// @Success 200 {object} dto.SwaggerEmailsResponse
// @Failure 400 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/emails [get]
func (h *Handler) listEmails(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil {
		respondError(c, domain.ErrInvalidRequest)
		return
	}
	emails, err := h.emailService.ListEmails(c.Request.Context(), c.Param("projectId"), c.Query("status"), limit)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Emails retrieved", Data: emails})
}

// emailStats godoc
// @Summary Summarise a project's delivery outcomes
// @Tags Email
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Success 200 {object} dto.SwaggerEmailStatsResponse
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/emails/stats [get]
func (h *Handler) emailStats(c *gin.Context) {
	stats, err := h.emailService.EmailStats(c.Request.Context(), c.Param("projectId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Email stats retrieved", Data: stats})
}

// listAudience godoc
// @Summary List a project's audience
// @Tags Audience
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Success 200 {object} dto.SwaggerAudienceResponse
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/audience [get]
func (h *Handler) listAudience(c *gin.Context) {
	audiences, err := h.projectService.ListAudience(c.Request.Context(), c.Param("projectId"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Audience retrieved", Data: audiences})
}

// removeAudience godoc
// @Summary Remove a recipient from a project's audience
// @Tags Audience
// @Security AdminKey
// @Produce json
// @Param projectId path string true "Project UUID"
// @Param id path string true "Audience UUID"
// @Success 200 {object} dto.Response
// @Failure 401 {object} dto.Response
// @Failure 404 {object} dto.Response
// @Failure 503 {object} dto.Response
// @Failure 500 {object} dto.Response
// @Router /api/projects/{projectId}/audience/{id} [delete]
func (h *Handler) removeAudience(c *gin.Context) {
	if err := h.projectService.RemoveAudience(c.Request.Context(), c.Param("projectId"), c.Param("id")); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ApiResponse[any]{Message: "Recipient removed", Data: nil})
}
