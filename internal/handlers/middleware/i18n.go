package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
)

const (
	// LanguageContextKey é a chave usada para armazenar o idioma no contexto do Gin
	LanguageContextKey = "language"
	// I18nServiceContextKey é a chave usada para armazenar o serviço i18n no contexto
	I18nServiceContextKey = "i18n_service"
)

// I18nMiddleware gerencia a detecção de idioma nas requisições
type I18nMiddleware struct {
	i18nService *i18n.Service
}

// NewI18nMiddleware cria um novo middleware de i18n
func NewI18nMiddleware(i18nService *i18n.Service) *I18nMiddleware {
	return &I18nMiddleware{
		i18nService: i18nService,
	}
}

// DetectLanguage detecta e configura o idioma da requisição
// Prioridade:
// 1. Query parameter ?lang=pt-BR (override explícito)
// 2. Accept-Language header (preferência do browser)
// 3. Idioma padrão (fallback)
func (m *I18nMiddleware) DetectLanguage() gin.HandlerFunc {
	return func(c *gin.Context) {
		var lang string

		// 1. Verificar query parameter
		if queryLang := c.Query("lang"); queryLang != "" {
			if m.i18nService.IsLanguageSupported(queryLang) {
				lang = queryLang
			}
		}

		// 2. Se não encontrou, verificar Accept-Language header
		if lang == "" {
			acceptLang := c.GetHeader("Accept-Language")
			lang = m.parseAcceptLanguage(acceptLang)
		}

		// 3. Se ainda não encontrou, usar idioma padrão
		if lang == "" {
			lang = m.i18nService.GetDefaultLanguage()
		}

		// Armazenar idioma e serviço no contexto
		c.Set(LanguageContextKey, lang)
		c.Set(I18nServiceContextKey, m.i18nService)

		c.Next()
	}
}

// parseAcceptLanguage analisa o header Accept-Language e retorna o melhor idioma suportado
// Exemplo: "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7" -> "pt-BR"
func (m *I18nMiddleware) parseAcceptLanguage(acceptLang string) string {
	if acceptLang == "" {
		return ""
	}

	// Dividir por vírgula para pegar todos os idiomas
	languages := strings.Split(acceptLang, ",")

	for _, lang := range languages {
		// Remover peso (;q=0.9) se existir
		lang = strings.TrimSpace(lang)
		if idx := strings.Index(lang, ";"); idx != -1 {
			lang = lang[:idx]
		}

		// Verificar se é suportado (exato)
		if m.i18nService.IsLanguageSupported(lang) {
			return lang
		}

		// Verificar variação sem região (pt-BR -> pt)
		if idx := strings.Index(lang, "-"); idx != -1 {
			baseLang := lang[:idx]
			if m.i18nService.IsLanguageSupported(baseLang) {
				return baseLang
			}
		}
	}

	return ""
}
