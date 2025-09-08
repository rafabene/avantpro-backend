package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/moogar0880/problems"
)

// JWTClaims representa as claims do JWT com validação aprimorada
type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware cria um middleware de autenticação JWT
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Obter token do cabeçalho Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenString == "" {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Authentication Required"
			problem.Detail = "Authorization token is required via header"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Analisar e validar o token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Validar o método de assinatura
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Invalid Token"
			problem.Detail = "Failed to parse JWT token"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Verificar se o token é válido
		if !token.Valid {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Invalid Token"
			problem.Detail = "JWT token is not valid"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Extrair claims
		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Invalid Token Claims"
			problem.Detail = "Failed to extract user information from token"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Analisar ID do usuário como UUID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Invalid User ID"
			problem.Detail = "User ID in token is not a valid UUID"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Definir o ID do usuário no contexto
		c.Set("userID", userID)
		c.Next()
	}
}

// GetUserIDFromContext extrai o ID do usuário do contexto Gin
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}

	userIDUUID, ok := userID.(uuid.UUID)
	return userIDUUID, ok
}

// GetUserIDFromHeader extrai e valida o cabeçalho User-ID
func GetUserIDFromHeader(c *gin.Context) (uuid.UUID, error) {
	userIDHeader := c.GetHeader("User-ID")
	if userIDHeader == "" {
		return uuid.Nil, errors.New("User-ID header is required")
	}

	userID, err := uuid.Parse(userIDHeader)
	if err != nil {
		return uuid.Nil, errors.New("invalid User-ID format")
	}

	return userID, nil
}
