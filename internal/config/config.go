package config

import (
	"os"
	"strings"
)

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

type Config struct {
	Environment Environment
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
}

type JWTConfig struct {
	Secret string
}

type ServerConfig struct {
	Port           string
	GinMode        string
	TrustedProxies []string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}


func LoadConfig() *Config {
	env := getEnvironment()

	config := &Config{
		Environment: env,
		Server: ServerConfig{
			Port:           getEnvWithDefault("PORT", "8080"),
			GinMode:        getGinMode(env),
			TrustedProxies: getTrustedProxies(),
		},
		Database: DatabaseConfig{
			Host:     getEnvWithDefault("DB_HOST", "localhost"),
			Port:     getEnvWithDefault("DB_PORT", "5432"),
			User:     getEnvWithDefault("DB_USER", "postgres"),
			Password: getEnvWithDefault("DB_PASSWORD", "postgres"),
			Name:     getEnvWithDefault("DB_NAME", "avantpro_backend"),
			SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret: getEnvWithDefault("JWT_SECRET", "your-secret-key-change-in-production"),
		},
	}

	return config
}

func getEnvironment() Environment {
	env := strings.ToLower(getEnvWithDefault("ENV", "development"))
	switch env {
	case "production", "prod":
		return Production
	default:
		return Development
	}
}

func getGinMode(env Environment) string {
	switch env {
	case Production:
		return "release"
	default:
		return "debug"
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == Development
}

func (c *Config) IsProduction() bool {
	return c.Environment == Production
}

func getTrustedProxies() []string {
	proxies := getEnvWithDefault("TRUSTED_PROXIES", "")
	if proxies == "" {
		return nil // No trusted proxies by default
	}
	
	// Split comma-separated list
	var result []string
	for _, proxy := range strings.Split(proxies, ",") {
		proxy = strings.TrimSpace(proxy)
		if proxy != "" {
			result = append(result, proxy)
		}
	}
	return result
}