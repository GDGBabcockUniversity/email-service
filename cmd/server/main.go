package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/app"
	"github.com/GDGBabcockUnivsersity/email_service/internal/infrastructure/config"
)

// @title GDG Email Service API
// @version 1.0
// @description Project and template management with authenticated asynchronous email delivery.
// @BasePath /
// @securityDefinitions.apikey AdminKey
// @in header
// @name X-Admin-Key
// @securityDefinitions.apikey ProjectApiKey
// @in header
// @name x-gdg_email-apikey
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := app.Run(ctx, cfg); err != nil {
		slog.Error("Email service stopped", "error", err)
		os.Exit(1)
	}
}
