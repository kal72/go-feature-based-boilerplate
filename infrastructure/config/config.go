package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration structure for the application.
// All values are loaded from environment variables or configuration files.
type Config struct {
	App           AppConfig           `mapstructure:",squash"`
	Server        ServerConfig        `mapstructure:",squash"`
	Database      DatabaseConfig      `mapstructure:",squash"`
	Mongo         MongoConfig         `mapstructure:",squash"`
	Redis         RedisConfig         `mapstructure:",squash"`
	JWT           JWTConfig           `mapstructure:",squash"`
	Logger        LoggerConfig        `mapstructure:",squash"`
	Telemetry     TelemetryConfig     `mapstructure:",squash"`
	CompanyClient CompanyClientConfig `mapstructure:",squash"`
}

// AppConfig holds general application metadata.
type AppConfig struct {
	Name        string `mapstructure:"APP_NAME"`
	Environment string `mapstructure:"APP_ENV"` // development | staging | production
	Version     string `mapstructure:"APP_VERSION"`
}

// ServerConfig holds gRPC and HTTP gateway listen addresses.
type ServerConfig struct {
	GRPCPort        int           `mapstructure:"SERVER_GRPC_PORT"`
	HTTPPort        int           `mapstructure:"SERVER_HTTP_PORT"`
	ReadTimeout     time.Duration `mapstructure:"SERVER_READ_TIMEOUT"`
	WriteTimeout    time.Duration `mapstructure:"SERVER_WRITE_TIMEOUT"`
	ShutdownTimeout time.Duration `mapstructure:"SERVER_SHUTDOWN_TIMEOUT"`
	DefaultTimeout  time.Duration `mapstructure:"SERVER_DEFAULT_TIMEOUT"`
}

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host            string        `mapstructure:"DB_HOST"`
	Port            int           `mapstructure:"DB_PORT"`
	Name            string        `mapstructure:"DB_NAME"`
	User            string        `mapstructure:"DB_USER"`
	Password        string        `mapstructure:"DB_PASSWORD"`
	SSLMode         string        `mapstructure:"DB_SSL_MODE"`
	MaxOpenConns    int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `mapstructure:"DB_CONN_MAX_LIFETIME"`
}

// DSN returns a PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// MongoConfig holds MongoDB connection parameters.
type MongoConfig struct {
	URI      string `mapstructure:"MONGO_URI"`
	Database string `mapstructure:"MONGO_DATABASE"`
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Addr     string `mapstructure:"REDIS_ADDR"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
}

// JWTConfig holds JWT signing configuration.
type JWTConfig struct {
	SecretKey       string        `mapstructure:"JWT_SECRET_KEY"`
	AccessTokenTTL  time.Duration `mapstructure:"JWT_ACCESS_TOKEN_TTL"`
	RefreshTokenTTL time.Duration `mapstructure:"JWT_REFRESH_TOKEN_TTL"`
}

// LoggerConfig holds logger settings.
type LoggerConfig struct {
	Level string `mapstructure:"LOG_LEVEL"` // debug | info | warn | error
}

// TelemetryConfig holds OpenTelemetry settings.
type TelemetryConfig struct {
	ServiceName  string `mapstructure:"OTEL_SERVICE_NAME"`
	OTLPEndpoint string `mapstructure:"OTEL_ENDPOINT"`
	Enabled      bool   `mapstructure:"OTEL_ENABLED"`
}

// CompanyClientConfig holds connection settings for the external Company gRPC service.
type CompanyClientConfig struct {
	GRPCAddr string        `mapstructure:"COMPANY_SERVICE_GRPC_ADDR"`
	Timeout  time.Duration `mapstructure:"COMPANY_SERVICE_TIMEOUT"`
}

// Load reads configuration using Viper from environment variables and optional .env file.
func Load() (*Config, error) {
	v := viper.New()

	// Read from .env file if present in current directory or parent directories
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("../")
	v.AddConfigPath("../../")
	_ = v.ReadInConfig()

	// Enable environment variables overriding
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	// Set safe defaults for timeout settings if not explicitly configured
	if cfg.Server.DefaultTimeout <= 0 {
		cfg.Server.DefaultTimeout = 15 * time.Second
	}
	if cfg.CompanyClient.Timeout <= 0 {
		cfg.CompanyClient.Timeout = 5 * time.Second
	}
	if cfg.CompanyClient.GRPCAddr == "" {
		cfg.CompanyClient.GRPCAddr = "localhost:50052"
	}

	return &cfg, nil
}
