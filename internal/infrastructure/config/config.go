package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config contém todas as configurações da aplicação
type Config struct {
	Env      string
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	OAuth    OAuthConfig
	SMTP     SMTPConfig
	Logging  LoggingConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port    string
	Host    string
	BaseURL string // URL base da API para construir URIs RFC 7807
}

type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
	MaxConns    int
	MinConns    int
	MaxIdleTime int
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  string
	RefreshExpiry string
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	RedirectURL        string
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

type LoggingConfig struct {
	Level string
}

type CORSConfig struct {
	AllowedOrigins string
}

// Load carrega as configurações do arquivo .env
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	config := &Config{
		Env: viper.GetString("ENV"),
		Server: ServerConfig{
			Port:    viper.GetString("PORT"),
			Host:    viper.GetString("HOST"),
			BaseURL: viper.GetString("API_BASE_URL"),
		},
		Database: DatabaseConfig{
			Host:        viper.GetString("DB_HOST"),
			Port:        viper.GetInt("DB_PORT"),
			User:        viper.GetString("DB_USER"),
			Password:    viper.GetString("DB_PASS"),
			DBName:      viper.GetString("DB_NAME"),
			SSLMode:     viper.GetString("DB_SSL_MODE"),
			MaxConns:    viper.GetInt("DB_MAX_CONNS"),
			MinConns:    viper.GetInt("DB_MIN_CONNS"),
			MaxIdleTime: viper.GetInt("DB_MAX_IDLE_TIME"),
		},
		Redis: RedisConfig{
			URL: viper.GetString("REDIS_URL"),
		},
		JWT: JWTConfig{
			Secret:        viper.GetString("JWT_SECRET"),
			AccessExpiry:  viper.GetString("JWT_ACCESS_EXPIRY"),
			RefreshExpiry: viper.GetString("JWT_REFRESH_EXPIRY"),
		},
		OAuth: OAuthConfig{
			GoogleClientID:     viper.GetString("GOOGLE_CLIENT_ID"),
			GoogleClientSecret: viper.GetString("GOOGLE_CLIENT_SECRET"),
			GitHubClientID:     viper.GetString("GITHUB_CLIENT_ID"),
			GitHubClientSecret: viper.GetString("GITHUB_CLIENT_SECRET"),
			RedirectURL:        viper.GetString("OAUTH_REDIRECT_URL"),
		},
		SMTP: SMTPConfig{
			Host:     viper.GetString("SMTP_HOST"),
			Port:     viper.GetInt("SMTP_PORT"),
			User:     viper.GetString("SMTP_USER"),
			Password: viper.GetString("SMTP_PASS"),
		},
		Logging: LoggingConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
		CORS: CORSConfig{
			AllowedOrigins: viper.GetString("CORS_ALLOWED_ORIGINS"),
		},
	}

	return config, nil
}

// DSN retorna a connection string do PostgreSQL
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}
