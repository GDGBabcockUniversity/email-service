package config

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate_MissingCredentials_ReturnsError(t *testing.T) {
	assert.Error(t, (&Config{}).Validate())
}

func TestDatabaseURL_EscapesCredentialsAndUsesConfiguredPort(t *testing.T) {
	cfg := &Config{DBHost: "localhost", DBPort: "55432", DBUser: "test", DBPassword: "p@ss:/? word", DBName: "email_service", DBSSLMode: "require"}
	value, err := url.Parse(cfg.DatabaseURL())
	require.NoError(t, err)
	password, _ := value.User.Password()
	assert.Equal(t, cfg.DBPassword, password)
	assert.Equal(t, "55432", value.Port())
	assert.Equal(t, "require", value.Query().Get("sslmode"))
	assert.Equal(t, "UTC", value.Query().Get("timezone"))
}

// Hosts inject PORT as a bare number, which is not a listen address on its own.
func TestValidate_NormalisesPortIntoAListenAddress(t *testing.T) {
	for value, expected := range map[string]string{
		"8080":         ":8080",
		":9000":        ":9000",
		"0.0.0.0:8080": "0.0.0.0:8080",
		" 3000 ":       ":3000",
		"":             ":8080",
	} {
		cfg := &Config{
			AdminApiKey: "admin", DBPassword: "secret", DBPort: "5432", DBSSLMode: "disable",
			SMTPHost: "smtp.example.com", SMTPPort: "587", SMTPFrom: "gdg@example.com",
			SMTPName: "GDG Babcock", SMTPUsername: "user", SMTPPassword: "pass",
			SMTPTLSMode: "starttls", UnsubscribeSecret: "unsub", PublicBaseURL: "https://mail.example.com",
			Port: value,
		}
		require.NoError(t, cfg.Validate(), "PORT=%q", value)
		assert.Equal(t, expected, cfg.Port, "PORT=%q", value)
	}
}
