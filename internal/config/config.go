package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment representa o ambiente de execução da aplicação
type Environment string

const (
	Development Environment = "development" // Ambiente de desenvolvimento
	Production  Environment = "production"  // Ambiente de produção
)

// Config contém todas as configurações da aplicação
type Config struct {
	Environment Environment    // Ambiente atual
	Server      ServerConfig   // Configurações do servidor
	Database    DatabaseConfig // Configurações do banco de dados
	JWT         JWTConfig      // Configurações do JWT
	Auth        AuthConfig     // Configurações de autenticação
	CORS        CORSConfig     // Configurações de CORS
}

// JWTConfig contém configurações relacionadas ao JWT
type JWTConfig struct {
	Secret                string // Chave secreta para assinar tokens JWT
	ExpirationHours       int    // Duração de expiração do token em horas
	RefreshExpirationDays int    // Duração de expiração do refresh token em dias
}

// AuthConfig contém configurações de autenticação e segurança
type AuthConfig struct {
	MaxLoginAttempts       int           // Máximo de tentativas de login antes do bloqueio
	AccountLockoutDuration time.Duration // Duração do bloqueio da conta após exceder tentativas
}

// ServerConfig contém configurações do servidor HTTP
type ServerConfig struct {
	Port           string   // Porta onde o servidor será executado
	GinMode        string   // Modo do Gin (debug/release)
	TrustedProxies []string // Lista de proxies confiáveis
}

// DatabaseConfig contém configurações de conexão com o banco de dados
type DatabaseConfig struct {
	Host     string // Host do banco de dados
	Port     string // Porta do banco de dados
	User     string // Usuário do banco de dados
	Password string // Senha do banco de dados
	Name     string // Nome do banco de dados
	SSLMode  string // Modo SSL para conexão
}

// CORSConfig contém configurações de CORS
type CORSConfig struct {
	AllowOrigins []string // Lista de origens permitidas
	AllowMethods []string // Métodos HTTP permitidos
	AllowHeaders []string // Headers permitidos
}

// LoadConfig carrega todas as configurações a partir das variáveis de ambiente
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
			Secret:                getEnvWithDefault("JWT_SECRET", "random-256-bit-key-here-change-in-production-f8b2c4e6a8d0f1e3b5c7a9d2e4f6b8c0a2d4e6f8b0c2a4e6f8d0b2c4a6e8f0d2"),
			ExpirationHours:       getIntWithDefault("JWT_EXPIRATION_HOURS", 24),
			RefreshExpirationDays: getIntWithDefault("JWT_REFRESH_EXPIRATION_DAYS", 30),
		},
		Auth: AuthConfig{
			MaxLoginAttempts:       getIntWithDefault("MAX_LOGIN_ATTEMPTS", 3),
			AccountLockoutDuration: time.Duration(getIntWithDefault("ACCOUNT_LOCKOUT_DURATION_MINUTES", 15)) * time.Minute,
		},
		CORS: CORSConfig{
			AllowOrigins: getCORSOrigins(),
			AllowMethods: getCORSMethods(),
			AllowHeaders: getCORSHeaders(),
		},
	}

	return config
}

// getEnvironment determina o ambiente de execução a partir da variável ENV
func getEnvironment() Environment {
	env := strings.ToLower(getEnvWithDefault("ENV", "development"))
	switch env {
	case "production", "prod":
		return Production
	default:
		return Development
	}
}

// getGinMode retorna o modo apropriado do Gin baseado no ambiente
func getGinMode(env Environment) string {
	switch env {
	case Production:
		return "release" // Modo release para produção
	default:
		return "debug" // Modo debug para desenvolvimento
	}
}

// getEnvWithDefault obtém uma variável de ambiente ou retorna um valor padrão
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntWithDefault obtém uma variável de ambiente como inteiro ou retorna um valor padrão
func getIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// IsDevelopment verifica se a aplicação está rodando em ambiente de desenvolvimento
func (c *Config) IsDevelopment() bool {
	return c.Environment == Development
}

// IsProduction verifica se a aplicação está rodando em ambiente de produção
func (c *Config) IsProduction() bool {
	return c.Environment == Production
}

// getTrustedProxies obtém a lista de proxies confiáveis da variável de ambiente
func getTrustedProxies() []string {
	proxies := getEnvWithDefault("TRUSTED_PROXIES", "")
	if proxies == "" {
		return nil // Nenhum proxy confiável por padrão
	}

	// Dividir lista separada por vírgulas
	var result []string
	for _, proxy := range strings.Split(proxies, ",") {
		proxy = strings.TrimSpace(proxy)
		if proxy != "" {
			result = append(result, proxy)
		}
	}
	return result
}

// getCORSOrigins obtém a lista de origens permitidas para CORS
func getCORSOrigins() []string {
	origins := getEnvWithDefault("CORS_ALLOW_ORIGINS", "http://localhost:4200,http://localhost:4201")

	var result []string
	for _, origin := range strings.Split(origins, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			result = append(result, origin)
		}
	}
	return result
}

// getCORSMethods obtém a lista de métodos HTTP permitidos para CORS
func getCORSMethods() []string {
	methods := getEnvWithDefault("CORS_ALLOW_METHODS", "GET,POST,PUT,DELETE,OPTIONS")

	var result []string
	for _, method := range strings.Split(methods, ",") {
		method = strings.TrimSpace(method)
		if method != "" {
			result = append(result, method)
		}
	}
	return result
}

// getCORSHeaders obtém a lista de headers permitidos para CORS
func getCORSHeaders() []string {
	headers := getEnvWithDefault("CORS_ALLOW_HEADERS", "Origin,Content-Type,Accept,Authorization,User-ID,Organization-ID")

	var result []string
	for _, header := range strings.Split(headers, ",") {
		header = strings.TrimSpace(header)
		if header != "" {
			result = append(result, header)
		}
	}
	return result
}
