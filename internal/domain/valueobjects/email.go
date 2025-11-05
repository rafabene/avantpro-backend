package valueobjects

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidEmail = errors.New("invalid email format")
)

// Email é um value object que garante que emails sejam sempre válidos
type Email struct {
	value string
}

// NewEmail cria um novo Email validado
func NewEmail(email string) (Email, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if !isValidEmail(email) {
		return Email{}, ErrInvalidEmail
	}

	return Email{value: email}, nil
}

// String retorna o valor do email
func (e Email) String() string {
	return e.value
}

// isValidEmail valida o formato do email
func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	pattern := `^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}
