package config_test

import (
	"os"
	"testing"
	"time"

	"go-feature-based-boilerplate/infrastructure/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might interfere
	os.Unsetenv("APP_NAME")
	os.Unsetenv("DB_HOST")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.NotEmpty(t, cfg.App.Name)
	assert.NotEmpty(t, cfg.Database.Host)
	assert.Equal(t, 50051, cfg.Server.GRPCPort)
	assert.Equal(t, 30*time.Second, cfg.Server.ReadTimeout)
}

func TestLoad_DotEnv(t *testing.T) {
	cfg, err := config.Load()
	require.NoError(t, err)
	// .env has DB_USER=postgres
	assert.Equal(t, "postgres", cfg.Database.User)
	assert.Equal(t, "postgres", cfg.Database.Password)
}

func TestLoad_EnvOverride(t *testing.T) {
	os.Setenv("APP_NAME", "custom-app")
	os.Setenv("SERVER_GRPC_PORT", "9090")
	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("SERVER_GRPC_PORT")
	}()

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "custom-app", cfg.App.Name)
	assert.Equal(t, 9090, cfg.Server.GRPCPort)
}

func TestDatabaseConfig_DSN(t *testing.T) {
	dbCfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpassword",
		Name:     "testdb",
		SSLMode:  "disable",
	}

	expectedDSN := "host=localhost port=5432 user=testuser password=testpassword dbname=testdb sslmode=disable"
	assert.Equal(t, expectedDSN, dbCfg.DSN())
}
