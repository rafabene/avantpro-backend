package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/moogar0880/problems"
)

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware creates a JWT authentication middleware
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Authentication Required"
			problem.Detail = "Authorization header is required"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Invalid Authorization Header"
			problem.Detail = "Authorization header must start with 'Bearer '"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Missing Token"
			problem.Detail = "Bearer token is required"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Parse and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
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

		// Check if the token is valid
		if !token.Valid {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Invalid Token"
			problem.Detail = "JWT token is not valid"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Invalid Token Claims"
			problem.Detail = "Failed to extract user information from token"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Set the user ID in the context
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

// GetUserIDFromContext extracts the user ID from the Gin context
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	
	userIDStr, ok := userID.(string)
	return userIDStr, ok
}