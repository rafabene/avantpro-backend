package i18n

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// setupTestLocales cria arquivos de locale temporários para testes
func setupTestLocales(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Criar arquivo en.json
	enContent := `{
  "welcome": "Welcome, {{.Name}}!",
  "user_created": "User created successfully",
  "error.user_not_found": "User not found"
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "en.json"), []byte(enContent), 0644); err != nil { //nolint:gosec
		t.Fatalf("failed to create en.json: %v", err)
	}

	// Criar arquivo pt-BR.json
	ptContent := `{
  "welcome": "Bem-vindo, {{.Name}}!",
  "user_created": "Usuário criado com sucesso",
  "error.user_not_found": "Usuário não encontrado"
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "pt-BR.json"), []byte(ptContent), 0644); err != nil { //nolint:gosec
		t.Fatalf("failed to create pt-BR.json: %v", err)
	}

	// Criar arquivo es.json
	esContent := `{
  "welcome": "¡Bienvenido, {{.Name}}!",
  "user_created": "Usuario creado exitosamente",
  "error.user_not_found": "Usuario no encontrado"
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "es.json"), []byte(esContent), 0644); err != nil { //nolint:gosec
		t.Fatalf("failed to create es.json: %v", err)
	}

	return tmpDir
}

func TestNewService(t *testing.T) {
	t.Run("carrega traduções com sucesso", func(t *testing.T) {
		tmpDir := setupTestLocales(t)

		service, err := NewService(tmpDir, "en")
		if err != nil {
			t.Fatalf("esperava sucesso, obteve erro: %v", err)
		}

		if service.GetDefaultLanguage() != "en" {
			t.Errorf("esperava idioma padrão 'en', obteve '%s'", service.GetDefaultLanguage())
		}

		supportedLangs := service.GetSupportedLanguages()
		if len(supportedLangs) != 3 {
			t.Errorf("esperava 3 idiomas suportados, obteve %d", len(supportedLangs))
		}
	})

	t.Run("erro quando diretório não existe", func(t *testing.T) {
		_, err := NewService("/diretorio/inexistente", "en")
		if err == nil {
			t.Error("esperava erro, obteve sucesso")
		}
	})

	t.Run("erro quando idioma padrão não existe", func(t *testing.T) {
		tmpDir := setupTestLocales(t)

		_, err := NewService(tmpDir, "fr")
		if err == nil {
			t.Error("esperava erro para idioma padrão inexistente, obteve sucesso")
		}
	})
}

func TestService_T(t *testing.T) {
	tmpDir := setupTestLocales(t)
	service, err := NewService(tmpDir, "en")
	if err != nil {
		t.Fatalf("falha ao inicializar serviço: %v", err)
	}

	t.Run("traduz mensagem simples em inglês", func(t *testing.T) {
		result := service.T("en", "user_created")
		expected := "User created successfully"
		if result != expected {
			t.Errorf("esperava '%s', obteve '%s'", expected, result)
		}
	})

	t.Run("traduz mensagem simples em português", func(t *testing.T) {
		result := service.T("pt-BR", "user_created")
		expected := "Usuário criado com sucesso"
		if result != expected {
			t.Errorf("esperava '%s', obteve '%s'", expected, result)
		}
	})

	t.Run("traduz mensagem com parâmetros", func(t *testing.T) {
		result := service.T("en", "welcome", map[string]interface{}{"Name": "John"})
		expected := "Welcome, John!"
		if result != expected {
			t.Errorf("esperava '%s', obteve '%s'", expected, result)
		}
	})

	t.Run("traduz mensagem com parâmetros em português", func(t *testing.T) {
		result := service.T("pt-BR", "welcome", map[string]interface{}{"Name": "João"})
		expected := "Bem-vindo, João!"
		if result != expected {
			t.Errorf("esperava '%s', obteve '%s'", expected, result)
		}
	})

	t.Run("fallback para idioma padrão quando chave não existe no idioma solicitado", func(t *testing.T) {
		result := service.T("fr", "user_created")
		expected := "User created successfully" // Fallback para inglês
		if result != expected {
			t.Errorf("esperava '%s', obteve '%s'", expected, result)
		}
	})

	t.Run("retorna chave quando tradução não existe", func(t *testing.T) {
		result := service.T("en", "chave.inexistente")
		expected := "chave.inexistente"
		if result != expected {
			t.Errorf("esperava '%s', obteve '%s'", expected, result)
		}
	})
}

func TestService_IsLanguageSupported(t *testing.T) {
	tmpDir := setupTestLocales(t)
	service, err := NewService(tmpDir, "en")
	if err != nil {
		t.Fatalf("falha ao inicializar serviço: %v", err)
	}

	tests := []struct {
		lang     string
		expected bool
	}{
		{"en", true},
		{"pt-BR", true},
		{"es", true},
		{"fr", false},
		{"de", false},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			result := service.IsLanguageSupported(tt.lang)
			if result != tt.expected {
				t.Errorf("para idioma '%s', esperava %v, obteve %v", tt.lang, tt.expected, result)
			}
		})
	}
}

func TestService_ThreadSafety(t *testing.T) {
	tmpDir := setupTestLocales(t)
	service, err := NewService(tmpDir, "en")
	if err != nil {
		t.Fatalf("falha ao inicializar serviço: %v", err)
	}

	// Executar traduções concorrentemente
	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			_ = service.T("en", "welcome", map[string]interface{}{"Name": "Test"})
		}()

		go func() {
			defer wg.Done()
			_ = service.T("pt-BR", "user_created")
		}()

		go func() {
			defer wg.Done()
			_ = service.IsLanguageSupported("en")
		}()
	}

	// Se houver race condition, este teste falhará com -race flag
	wg.Wait()
}
