package http

import (
	errs "errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rafabene/avantpro-backend/internal/domain/errors"
	"github.com/rafabene/avantpro-backend/internal/handlers/dto"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// UserHandler lida com requisições HTTP relacionadas a usuários
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler cria um novo UserHandler
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// CreateUser cria um novo usuário
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response := dto.ValidationErrorResponseI18n(c, nil)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// TODO: Implementar lógica completa
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Create user endpoint - implementation pending",
	})
}

// GetUser busca um usuário por ID
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		if errs.Is(err, errors.ErrUserNotFound) {
			response := dto.NotFoundErrorResponseI18n(c, "User")
			c.JSON(http.StatusNotFound, response)
			return
		}
		response := dto.InternalErrorResponseI18n(c)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	c.JSON(http.StatusOK, dto.ToUserResponse(user))
}

// ListUsers lista usuários
func (h *UserHandler) ListUsers(c *gin.Context) {
	// TODO: Implementar paginação e filtros
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "List users endpoint - implementation pending",
	})
}
