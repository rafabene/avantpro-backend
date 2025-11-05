package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"
)

// Service gerencia traduções e internacionalização
type Service struct {
	mu              sync.RWMutex
	translations    map[string]map[string]string // [language][key]message
	defaultLanguage string
}

// NewService cria um novo serviço de i18n
// localesDir: diretório contendo os arquivos JSON de tradução
// defaultLang: idioma padrão (fallback)
func NewService(localesDir, defaultLang string) (*Service, error) {
	s := &Service{
		translations:    make(map[string]map[string]string),
		defaultLanguage: defaultLang,
	}

	// Carregar todos os arquivos .json do diretório de locales
	files, err := filepath.Glob(filepath.Join(localesDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to find locale files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no locale files found in %s", localesDir)
	}

	// Carregar cada arquivo de tradução
	for _, file := range files {
		lang := filepath.Base(file)
		lang = lang[:len(lang)-5] // Remove extensão .json

		data, err := os.ReadFile(file) //nolint:gosec
		if err != nil {
			return nil, fmt.Errorf("failed to read locale file %s: %w", file, err)
		}

		var translations map[string]string
		if err := json.Unmarshal(data, &translations); err != nil {
			return nil, fmt.Errorf("failed to parse locale file %s: %w", file, err)
		}

		s.translations[lang] = translations
	}

	// Verificar se o idioma padrão existe
	if _, ok := s.translations[defaultLang]; !ok {
		return nil, fmt.Errorf("default language %s not found in locale files", defaultLang)
	}

	return s, nil
}

// T traduz uma chave para o idioma especificado
// Suporta interpolação de parâmetros usando templates Go ({{.Name}}, {{.Email}}, etc.)
func (s *Service) T(lang, key string, params ...map[string]interface{}) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Buscar tradução no idioma solicitado
	message := s.getTranslation(lang, key)

	// Se não encontrou, tentar idioma padrão
	if message == "" {
		message = s.getTranslation(s.defaultLanguage, key)
	}

	// Se ainda não encontrou, retornar a chave
	if message == "" {
		return key
	}

	// Se não há parâmetros, retornar mensagem diretamente
	if len(params) == 0 {
		return message
	}

	// Interpolar parâmetros usando template
	tmpl, err := template.New("msg").Parse(message)
	if err != nil {
		// Se houver erro no template, retornar mensagem sem interpolação
		return message
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, params[0]); err != nil {
		// Se houver erro na execução, retornar mensagem sem interpolação
		return message
	}

	return buf.String()
}

// getTranslation busca uma tradução sem lock (uso interno)
func (s *Service) getTranslation(lang, key string) string {
	if langMap, ok := s.translations[lang]; ok {
		if msg, ok := langMap[key]; ok {
			return msg
		}
	}
	return ""
}

// GetDefaultLanguage retorna o idioma padrão configurado
func (s *Service) GetDefaultLanguage() string {
	return s.defaultLanguage
}

// GetSupportedLanguages retorna lista de idiomas suportados
func (s *Service) GetSupportedLanguages() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	langs := make([]string, 0, len(s.translations))
	for lang := range s.translations {
		langs = append(langs, lang)
	}
	return langs
}

// IsLanguageSupported verifica se um idioma é suportado
func (s *Service) IsLanguageSupported(lang string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.translations[lang]
	return ok
}
