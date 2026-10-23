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
