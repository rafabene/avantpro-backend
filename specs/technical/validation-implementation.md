# Implementação de Validação - Value Objects

**Versão**: 1.0
**Data**: 06/11/2025

---

## 1. Visão Geral

Este documento descreve a implementação técnica dos **Value Objects** de validação no AvantPro Backend.

Value Objects são objetos imutáveis que encapsulam validação e lógica de domínio. São usados para garantir que dados inválidos nunca entrem no sistema.

**Localização**: `internal/domain/valueobjects/`

---

## 2. Password Value Object

### 2.1 Implementação

```go
// internal/domain/valueobjects/password.go
package valueobjects

import (
    "errors"
    "regexp"

    "golang.org/x/crypto/bcrypt"
)

const (
    // Custo bcrypt (12 = ~300ms por hash em hardware moderno)
    bcryptCost = 12

    // Limites de tamanho
    minPasswordLength = 8
    maxPasswordLength = 72  // Limite do bcrypt
)

// Password representa uma senha validada e hasheada
type Password struct {
    hash string  // Hash bcrypt da senha
}

// NewPassword cria um Password a partir de texto plano
// Valida e gera hash automaticamente
func NewPassword(plaintext string) (Password, error) {
    // Validar tamanho
    if len(plaintext) < minPasswordLength || len(plaintext) > maxPasswordLength {
        return Password{}, errors.New("error.password_length_invalid")
    }

    // Validar complexidade (1 letra + 1 número)
    hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(plaintext)
    hasNumber := regexp.MustCompile(`[0-9]`).MatchString(plaintext)

    if !hasLetter || !hasNumber {
        return Password{}, errors.New("error.password_weak")
    }

    // Gerar hash bcrypt
    hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
    if err != nil {
        return Password{}, err
    }

    return Password{hash: string(hash)}, nil
}

// NewPasswordFromHash reconstrói Password a partir de hash existente
// Usado ao carregar do banco de dados
func NewPasswordFromHash(hash string) Password {
    return Password{hash: hash}
}

// Verify verifica se uma senha em texto plano corresponde ao hash
func (p Password) Verify(plaintext string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(plaintext))
    return err == nil
}

// Hash retorna o hash bcrypt para armazenamento
func (p Password) Hash() string {
    return p.hash
}

// String oculta o hash por segurança
func (p Password) String() string {
    return "***"
}
```

### 2.2 Uso

```go
// Na camada de Service
func (s *UserService) Register(email, password string) error {
    // Criar value object (valida automaticamente)
    pwd, err := valueobjects.NewPassword(password)
    if err != nil {
        return err  // error.password_weak ou error.password_length_invalid
    }

    // Criar usuário
    user := entities.NewUser(email, pwd)

    return s.userRepo.Save(user)
}

// Na camada de Repository (persistir)
func (r *UserRepository) Save(user *entities.User) error {
    model := &UserModel{
        ID:           user.ID,
        Email:        user.Email.Value(),
        PasswordHash: user.Password.Hash(),  // Armazena hash
    }

    return r.db.Create(model).Error
}

// Na camada de Repository (carregar)
func (r *UserRepository) FindByEmail(email string) (*entities.User, error) {
    var model UserModel
    err := r.db.Where("email = ?", email).First(&model).Error
    if err != nil {
        return nil, err
    }

    // Reconstrói value object a partir do hash
    pwd := valueobjects.NewPasswordFromHash(model.PasswordHash)

    return entities.ReconstructUser(model.ID, model.Email, pwd), nil
}

// Verificar senha no login
func (s *AuthService) Login(email, password string) (*Token, error) {
    user, err := s.userRepo.FindByEmail(email)
    if err != nil {
        return nil, errors.New("invalid_credentials")
    }

    // Verifica senha
    if !user.Password.Verify(password) {
        return nil, errors.New("invalid_credentials")
    }

    return s.generateToken(user), nil
}
```

### 2.3 Testes

```go
// internal/domain/valueobjects/password_test.go
package valueobjects_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "avantpro-backend/internal/domain/valueobjects"
)

func TestNewPassword_Valid(t *testing.T) {
    tests := []string{
        "Senha123",
        "MyP@ssw0rd",
        "Abc12345",
        "Test1234",
    }

    for _, plaintext := range tests {
        pwd, err := valueobjects.NewPassword(plaintext)
        assert.NoError(t, err)
        assert.NotEmpty(t, pwd.Hash())
        assert.True(t, pwd.Verify(plaintext))
    }
}

func TestNewPassword_TooShort(t *testing.T) {
    pwd, err := valueobjects.NewPassword("Abc123")
    assert.Error(t, err)
    assert.Equal(t, "error.password_length_invalid", err.Error())
}

func TestNewPassword_NoLetter(t *testing.T) {
    pwd, err := valueobjects.NewPassword("12345678")
    assert.Error(t, err)
    assert.Equal(t, "error.password_weak", err.Error())
}

func TestNewPassword_NoNumber(t *testing.T) {
    pwd, err := valueobjects.NewPassword("senhaboa")
    assert.Error(t, err)
    assert.Equal(t, "error.password_weak", err.Error())
}

func TestPassword_VerifyCorrect(t *testing.T) {
    plaintext := "Senha123"
    pwd, _ := valueobjects.NewPassword(plaintext)

    assert.True(t, pwd.Verify(plaintext))
}

func TestPassword_VerifyIncorrect(t *testing.T) {
    pwd, _ := valueobjects.NewPassword("Senha123")

    assert.False(t, pwd.Verify("SenhaErrada"))
}
```

---

## 3. Email Value Object

### 3.1 Implementação

```go
// internal/domain/valueobjects/email.go
package valueobjects

import (
    "errors"
    "regexp"
    "strings"
)

// Regex RFC 5322 simplificado
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Lista de domínios descartáveis (pode ser carregada de arquivo/DB)
var disposableDomains = []string{
    "10minutemail.com",
    "guerrillamail.com",
    "mailinator.com",
    "tempmail.com",
    // Adicionar mais conforme necessário
}

// Email representa um endereço de email validado e normalizado
type Email struct {
    value string  // Email normalizado (lowercase, trimmed)
}

// NewEmail cria um Email validado e normalizado
func NewEmail(email string) (Email, error) {
    // Normalizar
    normalized := strings.ToLower(strings.TrimSpace(email))

    // Validar formato
    if !emailRegex.MatchString(normalized) {
        return Email{}, errors.New("error.invalid_email_format")
    }

    // Bloquear domínios descartáveis
    if isDisposableEmail(normalized) {
        return Email{}, errors.New("error.disposable_email_not_allowed")
    }

    return Email{value: normalized}, nil
}

// Value retorna o email normalizado
func (e Email) Value() string {
    return e.value
}

// String retorna o email para fmt.Print
func (e Email) String() string {
    return e.value
}

// Domain retorna a parte do domínio do email
func (e Email) Domain() string {
    parts := strings.Split(e.value, "@")
    if len(parts) == 2 {
        return parts[1]
    }
    return ""
}

// LocalPart retorna a parte local do email (antes do @)
func (e Email) LocalPart() string {
    parts := strings.Split(e.value, "@")
    if len(parts) == 2 {
        return parts[0]
    }
    return ""
}

// isDisposableEmail verifica se o domínio está na lista de descartáveis
func isDisposableEmail(email string) bool {
    domain := strings.Split(email, "@")[1]

    for _, disposable := range disposableDomains {
        if domain == disposable {
            return true
        }
    }

    return false
}
```

### 3.2 Uso

```go
// Criar email no registro
func (s *UserService) Register(emailStr, password string) error {
    // Valida e normaliza automaticamente
    email, err := valueobjects.NewEmail(emailStr)
    if err != nil {
        return err  // error.invalid_email_format ou error.disposable_email_not_allowed
    }

    // Verificar unicidade
    exists, _ := s.userRepo.ExistsByEmail(email.Value())
    if exists {
        return errors.New("error.email_already_exists")
    }

    pwd, _ := valueobjects.NewPassword(password)
    user := entities.NewUser(email, pwd)

    return s.userRepo.Save(user)
}

// Buscar por email
func (r *UserRepository) FindByEmail(emailStr string) (*entities.User, error) {
    // Normaliza antes de buscar
    email, err := valueobjects.NewEmail(emailStr)
    if err != nil {
        return nil, err
    }

    var model UserModel
    err = r.db.Where("email = ?", email.Value()).First(&model).Error

    // ...
}
```

### 3.3 Testes

```go
func TestNewEmail_Valid(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"user@example.com", "user@example.com"},
        {"User@Example.COM", "user@example.com"},  // Normalizado
        {"  test@test.com  ", "test@test.com"},    // Trimmed
        {"john.doe@company.co.uk", "john.doe@company.co.uk"},
    }

    for _, tt := range tests {
        email, err := valueobjects.NewEmail(tt.input)
        assert.NoError(t, err)
        assert.Equal(t, tt.expected, email.Value())
    }
}

func TestNewEmail_Invalid(t *testing.T) {
    tests := []struct {
        input       string
        expectedErr string
    }{
        {"invalid", "error.invalid_email_format"},
        {"@example.com", "error.invalid_email_format"},
        {"user@", "error.invalid_email_format"},
        {"user@10minutemail.com", "error.disposable_email_not_allowed"},
    }

    for _, tt := range tests {
        email, err := valueobjects.NewEmail(tt.input)
        assert.Error(t, err)
        assert.Equal(t, tt.expectedErr, err.Error())
    }
}

func TestEmail_Domain(t *testing.T) {
    email, _ := valueobjects.NewEmail("user@example.com")
    assert.Equal(t, "example.com", email.Domain())
}

func TestEmail_LocalPart(t *testing.T) {
    email, _ := valueobjects.NewEmail("user@example.com")
    assert.Equal(t, "user", email.LocalPart())
}
```

---

## 4. Integração com Entidades

### 4.1 User Entity

```go
// internal/domain/entities/user.go
package entities

import (
    "time"
    "avantpro-backend/internal/domain/valueobjects"
)

type User struct {
    ID        string
    Email     valueobjects.Email     // Value Object
    Password  valueobjects.Password  // Value Object
    Role      Role
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt *time.Time
}

// NewUser factory com validação embutida
func NewUser(email valueobjects.Email, password valueobjects.Password) *User {
    return &User{
        ID:        uuid.New().String(),
        Email:     email,
        Password:  password,
        Role:      RoleUser,  // Padrão
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
}

// ChangePassword valida nova senha
func (u *User) ChangePassword(newPassword valueobjects.Password) {
    u.Password = newPassword
    u.UpdatedAt = time.Now()
}
```

---

## 5. Benefícios da Abordagem

### ✅ Validação Centralizada
- Regras de validação em um único lugar
- Impossível criar senha/email inválido

### ✅ Type Safety
- Compilador garante que só senhas validadas são usadas
- Evita passar `string` onde deveria ser `Email`

### ✅ Imutabilidade
- Value Objects são imutáveis após criação
- Thread-safe por natureza

### ✅ Testabilidade
- Fácil de testar isoladamente
- Mocks não necessários (valores simples)

### ✅ Domain-Driven Design
- Linguagem ubíqua (Email, Password, não strings genéricos)
- Regras de negócio no domínio, não na infraestrutura

---

## 6. Mensagens de Erro i18n

### 6.1 Arquivo de Tradução

```json
// internal/infrastructure/i18n/locales/pt-BR.json
{
  "error.password_length_invalid": "Senha deve ter entre 8 e 72 caracteres",
  "error.password_weak": "Senha deve conter pelo menos 1 letra e 1 número",
  "error.invalid_email_format": "Formato de email inválido",
  "error.disposable_email_not_allowed": "Emails temporários não são permitidos",
  "error.email_already_exists": "Este email já está cadastrado"
}
```

### 6.2 Uso no Handler

```go
// internal/handlers/http/auth_handler.go
func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    err := h.authService.Register(req.Email, req.Password)
    if err != nil {
        // Traduz erro para idioma do request
        message := h.i18n.T(c, err.Error())

        c.JSON(400, gin.H{
            "error": err.Error(),        // Código do erro
            "message": message,          // Mensagem traduzida
        })
        return
    }

    c.JSON(201, gin.H{"status": "created"})
}
```

---

## 7. Referências

**Specs Relacionadas**:
- `specs/functional/user-registration.md` - Requisitos funcionais de validação
- `specs/technical/validation-i18n.md` - Validação e internacionalização

**Padrões**:
- Domain-Driven Design - Value Objects
- Clean Architecture - Domain layer puro
