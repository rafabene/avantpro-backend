package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	problemErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// OrganizationMembershipMiddleware valida se o usuário autenticado é membro da organização especificada no cabeçalho Organization-ID
func OrganizationMembershipMiddleware(orgService services.OrganizationServiceInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obter ID do usuário do contexto do token JWT (definido pelo AuthMiddleware)
		userID, exists := c.Get("userID")
		if !exists {
			prob := problemErrors.UnauthorizedError("User not authenticated", problemErrors.GetInstance(c))
			problemErrors.RespondWithProblem(c, prob)
			c.Abort()
			return
		}

		userIDUUID, ok := userID.(uuid.UUID)
		if !ok {
			prob := problemErrors.UnauthorizedError("Invalid user ID format", problemErrors.GetInstance(c))
			problemErrors.RespondWithProblem(c, prob)
			c.Abort()
			return
		}

		// Obter ID da organização do cabeçalho
		orgIDHeader := c.GetHeader("Organization-ID")
		if orgIDHeader == "" {
			prob := problemErrors.ValidationError("Organization-ID header is required", problemErrors.GetInstance(c))
			problemErrors.RespondWithProblem(c, prob)
			c.Abort()
			return
		}

		orgID, err := uuid.Parse(orgIDHeader)
		if err != nil {
			prob := problemErrors.ValidationError("Invalid Organization-ID format", problemErrors.GetInstance(c))
			problemErrors.RespondWithProblem(c, prob)
			c.Abort()
			return
		}

		// Validar se o usuário é membro da organização
		isMember, err := orgService.IsUserMemberOfOrganization(userIDUUID, orgID)
		if err != nil {
			if err.Error() == "organization not found" {
				prob := problemErrors.NotFoundError("Organization not found", problemErrors.GetInstance(c))
				problemErrors.RespondWithProblem(c, prob)
				c.Abort()
				return
			}
			prob := problemErrors.InternalError(problemErrors.GetInstance(c))
			problemErrors.RespondWithProblem(c, prob)
			c.Abort()
			return
		}

		if !isMember {
			prob := problemErrors.ForbiddenError("User is not a member of this organization", problemErrors.GetInstance(c))
			problemErrors.RespondWithProblem(c, prob)
			c.Abort()
			return
		}

		// Armazenar ID da organização no contexto para uso dos controllers
		c.Set("organizationID", orgID)
		c.Next()
	}
}

// GetOrganizationIDFromContext extrai o ID da organização do contexto gin
func GetOrganizationIDFromContext(c *gin.Context) (uuid.UUID, error) {
	orgID, exists := c.Get("organizationID")
	if !exists {
		return uuid.Nil, errors.New("organization ID not found in context")
	}

	orgIDUUID, ok := orgID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("invalid organization ID format in context")
	}

	return orgIDUUID, nil
}
