package dto

import (
	"github.com/gin-gonic/gin"

	"github.com/rafabene/avantpro-backend/internal/handlers/middleware"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
)

// T é um helper para traduzir mensagens no contexto do Gin
// Uso: dto.T(c, "welcome", map[string]interface{}{"Name": "John"})
func T(c *gin.Context, key string, params ...map[string]interface{}) string {
	// Buscar serviço i18n do contexto
	i18nService, exists := c.Get(middleware.I18nServiceContextKey)
	if !exists {
		// Fallback: retornar a chave se serviço não estiver disponível
		return key
	}

	service, ok := i18nService.(*i18n.Service)
	if !ok {
		return key
	}

	// Buscar idioma do contexto
	lang := GetLanguage(c)

	// Traduzir
	return service.T(lang, key, params...)
}

// GetLanguage retorna o idioma configurado no contexto da requisição
func GetLanguage(c *gin.Context) string {
	lang, exists := c.Get(middleware.LanguageContextKey)
	if !exists {
		return "en" // Fallback
	}

	langStr, ok := lang.(string)
	if !ok {
		return "en"
	}

	return langStr
}
