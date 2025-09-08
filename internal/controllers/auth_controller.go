package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	problemErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// AuthController gerencia requisições HTTP para operações de autenticação
type AuthController struct {
	authService services.AuthService
}

// Conversion functions
func (c *AuthController) toServiceLoginRequest(req *ServiceLoginRequest) *services.LoginRequest {
	return &services.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}
}

func (c *AuthController) toServiceRegisterRequest(req *ServiceRegisterRequest) *services.RegisterRequest {
	return &services.RegisterRequest{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	}
}

// NewAuthController cria uma nova instância do AuthController
func NewAuthController(authService services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

// getUserIDFromContext extrai o ID do usuário do contexto do token JWT
func (c *AuthController) getUserIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
	userID, exists := ctx.Get("userID")
	if !exists {
		return uuid.Nil, errors.New("usuário não autenticado")
	}

	userIDUUID, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.New("formato de ID de usuário inválido")
	}

	return userIDUUID, nil
}

// Login autentica um usuário e retorna um token
// @Summary Fazer login do usuário
// @Description Autentica usuário com email e senha
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body controllers.LoginRequest true "Credenciais de login"
// @Success 200 {object} controllers.LoginResponse "Resposta de sucesso com token e informações do usuário"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Formato JSON inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obtém informações do cliente para log de segurança
	clientIP := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	// Convert to service format
	modelReq := &ServiceLoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	response, err := c.authService.LoginWithContext(c.toServiceLoginRequest(modelReq), clientIP, userAgent)
	if err != nil {
		if err.Error() == "email ou senha incorretos" {
			prob := problemErrors.UnauthorizedError("Email ou senha incorretos", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		// Trata erros de bloqueio de conta
		if strings.Contains(err.Error(), "Conta bloqueada") {
			prob := problemErrors.UnauthorizedError(err.Error(), problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Register cria uma nova conta de usuário
// @Summary Registrar novo usuário
// @Description Criar uma nova conta de usuário
// @Tags auth
// @Accept json
// @Produce json
// @Param user body controllers.RegisterRequest true "Dados de registro"
// @Success 201 {object} controllers.LoginResponse "Usuário criado com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 409 {object} errors.ConflictProblem "Conflito"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /auth/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	var req RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Formato JSON inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Convert to service format
	modelReq := &ServiceRegisterRequest{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	response, err := c.authService.Register(c.toServiceRegisterRequest(modelReq))
	if err != nil {
		if err.Error() == problemErrors.ErrUsernameAlreadyExists {
			prob := problemErrors.ConflictError(err.Error(), problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// RequestPasswordReset envia email de redefinição de senha
// @Summary Solicitar redefinição de senha
// @Description Enviar email de redefinição de senha para o usuário
// @Tags auth
// @Accept json
// @Produce json
// @Param email body controllers.PasswordResetRequest true "Email para redefinição de senha"
// @Success 200 {object} controllers.MessageResponse "Email de redefinição enviado com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /auth/password-reset [post]
func (c *AuthController) RequestPasswordReset(ctx *gin.Context) {
	var req PasswordResetRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Formato JSON inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	err := c.authService.RequestPasswordReset(req.Email)
	if err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, MessageResponse{
		Message: "Email de redefinição de senha enviado com sucesso",
	})
}

// ResetPassword redefine a senha do usuário usando token
// @Summary Redefinir senha
// @Description Redefinir senha do usuário usando token de redefinição
// @Tags auth
// @Accept json
// @Produce json
// @Param reset body controllers.PasswordResetConfirmRequest true "Dados de redefinição de senha"
// @Success 200 {object} controllers.MessageResponse "Senha redefinida com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 404 {object} errors.NotFoundProblem "Não encontrado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /auth/password-reset/confirm [post]
func (c *AuthController) ResetPassword(ctx *gin.Context) {
	var req PasswordResetConfirmRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Formato JSON inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	err := c.authService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		if err.Error() == "invalid or expired token" {
			prob := problemErrors.BadRequestError("Token de redefinição inválido ou expirado", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, MessageResponse{
		Message: "Senha redefinida com sucesso",
	})
}

// UpdateLastSelectedOrganization atualiza a preferência da última organização selecionada do usuário
// @Summary Atualizar última organização selecionada
// @Description Atualizar preferência da última organização selecionada do usuário
// @Tags auth
// @Accept json
// @Produce json
// @Param organization body controllers.UpdateLastSelectedOrganizationRequest true "Dados de preferência da organização"
// @Success 200 {object} controllers.MessageResponse "Última organização selecionada atualizada com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /auth/last-selected-organization [put]
// @Security Bearer
func (c *AuthController) UpdateLastSelectedOrganization(ctx *gin.Context) {
	var req UpdateLastSelectedOrganizationRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Formato JSON inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID do usuário do contexto JWT
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	err = c.authService.UpdateLastSelectedOrganization(userID, req.OrganizationID)
	if err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, MessageResponse{
		Message: "Última organização selecionada atualizada com sucesso",
	})
}
