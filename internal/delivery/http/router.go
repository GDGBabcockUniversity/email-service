package http

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/GDGBabcockUnivsersity/email_service/internal/delivery/dto"
	"github.com/GDGBabcockUnivsersity/email_service/internal/domain"
	"github.com/GDGBabcockUnivsersity/email_service/internal/services"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/GDGBabcockUnivsersity/email_service/cmd/server/docs"
)

func NewRouter(adminKey string, projectService *services.ProjectService, templateService *services.TemplateService, apiKeyService *services.ApiKeyService, emailService *services.EmailService, sanityService *services.SanityService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	_ = router.SetTrustedProxies(nil)
	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json"))
	router.GET("/swagger/*any", swaggerHandler)
	handler := NewHandler(projectService, templateService, apiKeyService, emailService, sanityService)
	router.POST("/api/email", handler.authenticateEmail(), handler.sendEmail)

	router.POST("/sanity/webhook", handler.sanityWebhook)
	admin := router.Group("/api", authenticateAdmin(adminKey))
	admin.POST("/projects", handler.createProject)
	admin.GET("/projects", handler.listProjects)
	admin.GET("/projects/:projectId", handler.getProject)
	admin.PUT("/projects/:projectId", handler.updateProject)
	admin.DELETE("/projects/:projectId", handler.deleteProject)
	admin.POST("/projects/:projectId/templates", handler.createTemplate)
	admin.GET("/projects/:projectId/templates", handler.listTemplates)
	admin.GET("/projects/:projectId/templates/:id", handler.getTemplate)
	admin.PUT("/projects/:projectId/templates/:id", handler.updateTemplate)
	admin.DELETE("/projects/:projectId/templates/:id", handler.deleteTemplate)
	admin.POST("/projects/:projectId/api-keys", handler.createKey)
	admin.GET("/projects/:projectId/api-keys", handler.listKeys)
	admin.DELETE("/projects/:projectId/api-keys/:id", handler.revokeKey)
	admin.POST("/projects/:projectId/audience", handler.addAudience)
	admin.GET("/projects/:projectId/audience", handler.listAudience)
	admin.DELETE("/projects/:projectId/audience/:id", handler.removeAudience)
	admin.POST("/projects/:projectId/emails", handler.sendProjectEmail)
	admin.GET("/projects/:projectId/emails", handler.listEmails)
	admin.GET("/projects/:projectId/emails/stats", handler.emailStats)
	router.NoRoute(func(c *gin.Context) {
		respondError(c, domain.ErrNotFound)
	})
	return router
}

func authenticateAdmin(key string) gin.HandlerFunc {
	expected := sha256.Sum256([]byte(key))
	return func(c *gin.Context) {
		supplied := sha256.Sum256([]byte(c.GetHeader("X-Admin-Key")))
		if key == "" || subtle.ConstantTimeCompare(supplied[:], expected[:]) != 1 {
			respondError(c, domain.ErrUnauthorized)
			return
		}
		c.Next()
	}
}

func (h *Handler) authenticateEmail() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID, err := h.apiKeyService.Authenticate(c.Request.Context(), c.GetHeader("x-gdg_email-apikey"))
		if err != nil {
			respondError(c, err)
			return
		}
		c.Set("projectID", projectID)
		c.Next()
	}
}

func respondError(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "Internal Server Error"
	switch {
	case errors.Is(err, domain.ErrInvalidRequest):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		status, message = http.StatusUnauthorized, domain.ErrUnauthorized.Error()
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrProjectNotFound):
		status, message = http.StatusNotFound, "Resource Not Found"
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrDuplicateProjectName):
		status, message = http.StatusConflict, "Resource Already Exists"
	case errors.Is(err, domain.ErrUnavailable), errors.Is(err, context.DeadlineExceeded):
		status, message = http.StatusServiceUnavailable, domain.ErrUnavailable.Error()
	}
	c.Abort()
	c.JSON(status, dto.ApiResponse[any]{Message: message, Data: nil})
}
