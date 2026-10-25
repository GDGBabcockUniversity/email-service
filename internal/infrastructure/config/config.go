package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	REDIS_URL     string
	HTTPAddr      string
	AdminApiKey   string
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	SMTPFrom      string
	SMTPName      string
	SMTPTLSMode   string
	SanityUrl     string
	WebhookSecret string
}

func Load() (*Config, error) {
	cfg := &Config{
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", ""),
		DBName:        getEnv("DB_NAME", "email_service"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		REDIS_URL:     getEnv("REDIS_URL", "redis://localhost:6379/0"),
		HTTPAddr:      getEnv("HTTP_ADDR", ":8080"),
		AdminApiKey:   getEnv("ADMIN_API_KEY", ""),
		SMTPHost:      getEnv("SMTP_HOST", ""),
		SMTPPort:      getEnv("SMTP_PORT", "587"),
		SMTPUsername:  getEnv("SMTP_USERNAME", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:      getEnv("SMTP_FROM", ""),
		SMTPName:      getEnv("SMTP_NAME", ""),
		SMTPTLSMode:   getEnv("SMTP_TLS_MODE", "starttls"),
		SanityUrl:     getEnv("SANITY_URL", ""),
		WebhookSecret: getEnv("WEBHOOK_SECRET", ""),
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.AdminApiKey) == "" || c.DBPassword == "" || c.SMTPHost == "" || c.SMTPFrom == "" || c.SMTPName == "" {
		return fmt.Errorf("DB_PASSWORD, ADMIN_API_KEY, SMTP_HOST, SMTP_NAME and SMTP_FROM are required")
	}
	for _, port := range []string{c.DBPort, c.SMTPPort} {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return fmt.Errorf("invalid database or SMTP port")
		}
	}
	if _, _, err := net.SplitHostPort(c.HTTPAddr); err != nil {
		return fmt.Errorf("invalid HTTP_ADDR: %w", err)
	}
	if strings.ContainsAny(c.SMTPFrom, "\r\n") {
		return fmt.Errorf("SMTP_FROM must be a single email address")
	}
	switch c.SMTPTLSMode {
	case "starttls", "tls":
		if c.SMTPUsername == "" || c.SMTPPassword == "" {
			return fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD are required for authenticated SMTP")
		}
	case "none":
		if c.SMTPUsername != "" || c.SMTPPassword != "" {
			return fmt.Errorf("SMTP_TLS_MODE=none is only for unauthenticated local test servers")
		}
	default:
		return fmt.Errorf("SMTP_TLS_MODE must be starttls, tls or none")
	}
	switch c.DBSSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("invalid DB_SSLMODE")
	}
	return nil
}

func (c *Config) DatabaseURL() string {
	value := &url.URL{Scheme: "postgres", User: url.UserPassword(c.DBUser, c.DBPassword), Host: net.JoinHostPort(c.DBHost, c.DBPort), Path: "/" + c.DBName}
	query := value.Query()
	query.Set("sslmode", c.DBSSLMode)
	query.Set("timezone", "UTC")
	query.Set("connect_timeout", "5")
	value.RawQuery = query.Encode()
	return value.String()
}

func getEnv(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}
