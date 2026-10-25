package app

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"time"

	delivery "github.com/GDGBabcockUnivsersity/email_service/internal/delivery/http"
	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/config"
	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/postgres"
	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/redis"
	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/smtp"
	"github.com/GDGBabcockUnivsersity/email_service/internal/services"
)

func Run(ctx context.Context, cfg *config.Config) error {
	db, err := postgres.Connect(cfg)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	client, err := redis.Connect(cfg.REDIS_URL)
	if err != nil {
		return err
	}
	defer client.Close()
	queue := redis.NewQueue(client)
	startupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err = queue.Initialize(startupCtx)
	cancel()
	if err != nil {
		return err
	}

	projectRepo := postgres.NewProjectRepository(db)
	templateRepo := postgres.NewTemplateRepository(db)
	keyRepo := postgres.NewApiKeyRepository(db)
	emailRepo := postgres.NewEmailRepository(db)
	audienceRepo := postgres.NewAudienceRepository(db)
	unsubscribeTokens := services.NewUnsubscribeTokens(cfg.UnsubscribeSecret, cfg.PublicBaseURL)
	projectService := services.NewProjectService(projectRepo, audienceRepo, unsubscribeTokens, postgres.OwnerID)
	templateService := services.NewTemplateService(projectRepo, templateRepo)
	keyService := services.NewApiKeyService(keyRepo, projectRepo)
	emailService := services.NewEmailService(emailRepo, audienceRepo, templateRepo, queue, unsubscribeTokens)
	unsubscribeService := services.NewUnsubscribeService(audienceRepo, unsubscribeTokens)
	consumer := services.NewEmailConsumer(emailRepo, smtp.NewSender(cfg))
	webhookService := services.NewSanityService(cfg.WebhookSecret, cfg.WebhookSecret, emailService)
	router := delivery.NewRouter(cfg.AdminApiKey, projectService, templateService, keyService, emailService, webhookService, unsubscribeService)

	server := &http.Server{
		Addr:              cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      25 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	workerCtx, stopWorker := context.WithCancel(context.Background())
	workerDone := make(chan struct{})

	go func() {
		defer close(workerDone)
		consume(workerCtx, queue, consumer)
	}()
	serverError := make(chan error, 1)
	go func() {
		serverError <- server.ListenAndServe()
	}()

	slog.Info("Email service started", "address", cfg.Port)
	var runErr error
	select {
	case <-ctx.Done():
	case runErr = <-serverError:
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
	}
	stopWorker()
	<-workerDone
	if errors.Is(runErr, http.ErrServerClosed) {
		return nil
	}
	return runErr
}

func consume(ctx context.Context, queue *redis.Queue, consumer *services.EmailConsumer) {
	name := "worker-" + rand.Text()
	for ctx.Err() == nil {
		message, err := queue.Receive(ctx, name)

		if err == nil && message != nil {
			err = consumer.Process(ctx, message.EmailID)
			if err == nil {
				err = queue.Ack(ctx, message.ID)
			}
		}
		if err != nil && ctx.Err() == nil {
			slog.Error("Email worker operation failed; pending jobs will be reclaimed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
	}
}
