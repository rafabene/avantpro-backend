package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
)

func setupTestI18n(t *testing.T) *i18n.Service {
	t.Helper()

	tmpDir := t.TempDir()

	// Criar arquivo en.json
	enContent := `{"welcome": "Welcome"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "en.json"), []byte(enContent), 0644); err != nil { //nolint:gosec
		t.Fatalf("failed to create en.json: %v", err)
	}

	// Criar arquivo pt-BR.json
	ptContent := `{"welcome": "Bem-vindo"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "pt-BR.json"), []byte(ptContent), 0644); err != nil { //nolint:gosec
		t.Fatalf("failed to create pt-BR.json: %v", err)
	}

	// Criar arquivo es.json
	esContent := `{"welcome": "Bienvenido"}`
	if err := os.WriteFile(filepath.Join(tmpDir, "es.json"), []byte(esContent), 0644); err != nil { //nolint:gosec
		t.Fatalf("failed to create es.json: %v", err)
	}

	service, err := i18n.NewService(tmpDir, "en")
	if err != nil {
		t.Fatalf("failed to initialize i18n service: %v", err)
	}

	return service
}

func TestI18nMiddleware_DetectLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	i18nService := setupTestI18n(t)
	middleware := NewI18nMiddleware(i18nService)

	t.Run("detecta idioma do query parameter", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/?lang=pt-BR", nil)

		middleware.DetectLanguage()(c)

		lang, exists := c.Get(LanguageContextKey)
		if !exists {
			t.Fatal("idioma não foi definido no contexto")
		}

		if lang != "pt-BR" {
			t.Errorf("esperava 'pt-BR', obteve '%s'", lang)
		}
	})

	t.Run("detecta idioma do Accept-Language header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Language", "es,en;q=0.9")
		c.Request = req

		middleware.DetectLanguage()(c)

		lang, exists := c.Get(LanguageContextKey)
		if !exists {
			t.Fatal("idioma não foi definido no contexto")
		}

		if lang != "es" {
			t.Errorf("esperava 'es', obteve '%s'", lang)
		}
	})

	t.Run("usa idioma padrão quando nenhum é especificado", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware.DetectLanguage()(c)

		lang, exists := c.Get(LanguageContextKey)
		if !exists {
			t.Fatal("idioma não foi definido no contexto")
		}

		if lang != "en" {
			t.Errorf("esperava 'en' (padrão), obteve '%s'", lang)
		}
	})

	t.Run("query parameter tem prioridade sobre Accept-Language", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/?lang=pt-BR", nil)
		req.Header.Set("Accept-Language", "es")
		c.Request = req

		middleware.DetectLanguage()(c)

		lang, exists := c.Get(LanguageContextKey)
		if !exists {
			t.Fatal("idioma não foi definido no contexto")
		}

		if lang != "pt-BR" {
			t.Errorf("esperava 'pt-BR', obteve '%s'", lang)
		}
	})

	t.Run("ignora query parameter inválido e usa Accept-Language", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/?lang=fr", nil)
		req.Header.Set("Accept-Language", "es")
		c.Request = req

		middleware.DetectLanguage()(c)

		lang, exists := c.Get(LanguageContextKey)
		if !exists {
			t.Fatal("idioma não foi definido no contexto")
		}

		if lang != "es" {
			t.Errorf("esperava 'es', obteve '%s'", lang)
		}
	})

	t.Run("define serviço i18n no contexto", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)

		middleware.DetectLanguage()(c)

		service, exists := c.Get(I18nServiceContextKey)
		if !exists {
			t.Fatal("serviço i18n não foi definido no contexto")
		}

		if service == nil {
			t.Error("serviço i18n é nulo")
		}
	})
}

func TestI18nMiddleware_parseAcceptLanguage(t *testing.T) {
	i18nService := setupTestI18n(t)
	middleware := NewI18nMiddleware(i18nService)

	tests := []struct {
		name       string
		acceptLang string
		expected   string
	}{
		{
			name:       "idioma único suportado",
			acceptLang: "pt-BR",
			expected:   "pt-BR",
		},
		{
			name:       "múltiplos idiomas, primeiro é suportado",
			acceptLang: "es,pt-BR;q=0.9,en;q=0.8",
			expected:   "es",
		},
		{
			name:       "múltiplos idiomas, segundo é suportado",
			acceptLang: "fr,pt-BR;q=0.9,en;q=0.8",
			expected:   "pt-BR",
		},
		{
			name:       "nenhum idioma suportado",
			acceptLang: "fr,de;q=0.9",
			expected:   "",
		},
		{
			name:       "header vazio",
			acceptLang: "",
			expected:   "",
		},
		{
			name:       "idioma com região não suportado, mas base é",
			acceptLang: "pt",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := middleware.parseAcceptLanguage(tt.acceptLang)
			if result != tt.expected {
				t.Errorf("esperava '%s', obteve '%s'", tt.expected, result)
			}
		})
	}
}

func TestI18nMiddleware_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	i18nService := setupTestI18n(t)
	middleware := NewI18nMiddleware(i18nService)

	router := gin.New()
	router.Use(middleware.DetectLanguage())
	router.GET("/test", func(c *gin.Context) {
		lang, _ := c.Get(LanguageContextKey)
		service, _ := c.Get(I18nServiceContextKey)
		i18nSvc := service.(*i18n.Service)

		message := i18nSvc.T(lang.(string), "welcome")
		c.JSON(http.StatusOK, gin.H{"message": message})
	})

	t.Run("integração completa com português", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test?lang=pt-BR", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("esperava status 200, obteve %d", w.Code)
		}

		expected := `{"message":"Bem-vindo"}`
		if w.Body.String() != expected {
			t.Errorf("esperava '%s', obteve '%s'", expected, w.Body.String())
		}
	})

	t.Run("integração completa com espanhol via Accept-Language", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "es")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("esperava status 200, obteve %d", w.Code)
		}

		expected := `{"message":"Bienvenido"}`
		if w.Body.String() != expected {
			t.Errorf("esperava '%s', obteve '%s'", expected, w.Body.String())
		}
	})
}
