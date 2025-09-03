package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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
		var tokenString string

		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// For WebSocket connections, try to get token from query parameter
			authQuery := c.Query("authorization")
			if authQuery != "" && strings.HasPrefix(authQuery, "Bearer ") {
				tokenString = strings.TrimPrefix(authQuery, "Bearer ")
			}
		}

		if tokenString == "" {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Authentication Required"
			problem.Detail = "Authorization token is required (via header or query parameter)"
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

		// Parse user ID as UUID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			problem := problems.NewStatusProblem(http.StatusUnauthorized)
			problem.Title = "Invalid User ID"
			problem.Detail = "User ID in token is not a valid UUID"
			c.JSON(problem.Status, problem)
			c.Abort()
			return
		}

		// Set the user ID in the context
		c.Set("userID", userID)
		c.Next()
	}
}

// GetUserIDFromContext extracts the user ID from the Gin context
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		return uuid.Nil, false
	}

	userIDUUID, ok := userID.(uuid.UUID)
	return userIDUUID, ok
}
