package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	problemErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// OrganizationController gerencia requisições HTTP relacionadas à organização
type OrganizationController struct {
	service services.OrganizationServiceInterface
}

// Conversion functions
func (c *OrganizationController) toServiceOrganizationCreateRequest(req *ServiceOrganizationCreateRequest) *services.OrganizationCreateRequest {
	return &services.OrganizationCreateRequest{
		Name:        req.Name,
		Description: req.Description,
	}
}

func (c *OrganizationController) toServiceOrganizationUpdateRequest(req *ServiceOrganizationUpdateRequest) *services.OrganizationUpdateRequest {
	return &services.OrganizationUpdateRequest{
		Name:        req.Name,
		Description: req.Description,
	}
}

func (c *OrganizationController) toServiceOrganizationInviteRequest(req *ServiceOrganizationInviteRequest) *services.OrganizationInviteRequest {
	return &services.OrganizationInviteRequest{
		Email: req.Email,
		Role:  services.OrganizationRole(req.Role),
	}
}

func (c *OrganizationController) toServiceOrganizationMemberUpdateRequest(req *ServiceOrganizationMemberUpdateRequest) *services.OrganizationMemberUpdateRequest {
	return &services.OrganizationMemberUpdateRequest{
		Role: services.OrganizationRole(req.Role),
	}
}

// NewOrganizationController cria um novo controlador de organização
func NewOrganizationController(service services.OrganizationServiceInterface) *OrganizationController {
	return &OrganizationController{
		service: service,
	}
}

// getOrganizationIDFromPath extrai e valida o orgid do path
func (c *OrganizationController) getOrganizationIDFromPath(ctx *gin.Context) (uuid.UUID, error) {
	orgIDParam := ctx.Param("orgid")
	if orgIDParam == "" {
		return uuid.Nil, errors.New("parâmetro orgid é obrigatório")
	}

	orgID, err := uuid.Parse(orgIDParam)
	if err != nil {
		return uuid.Nil, errors.New("formato de orgid inválido")
	}

	return orgID, nil
}

// getOrganizationIDFromHeader extrai e valida o cabeçalho Organization-ID (mantido para compatibilidade)
func (c *OrganizationController) getOrganizationIDFromHeader(ctx *gin.Context) (uuid.UUID, error) {
	orgIDHeader := ctx.GetHeader("Organization-ID")
	if orgIDHeader == "" {
		return uuid.Nil, errors.New("cabeçalho Organization-ID é obrigatório")
	}

	orgID, err := uuid.Parse(orgIDHeader)
	if err != nil {
		return uuid.Nil, errors.New("formato de Organization-ID inválido")
	}

	return orgID, nil
}

// getUserIDFromContext extrai o ID do usuário do contexto do token JWT
func (c *OrganizationController) getUserIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
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

// CreateOrganization cria uma nova organização
// @Summary Criar uma nova organização
// @Description Criar uma nova organização com o usuário autenticado como administrador
// @Tags organizations
// @Accept json
// @Produce json
// @Param organization body controllers.OrganizationCreateRequest true "Dados da organização"
// @Success 201 {object} controllers.OrganizationResponse "Organização criada com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /organizations [post]
func (c *OrganizationController) CreateOrganization(ctx *gin.Context) {
	// Obter ID do usuário do token JWT
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	var req OrganizationCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	// Convert to service format
	modelReq := &ServiceOrganizationCreateRequest{
		Name:        req.Name,
		Description: req.Description,
	}

	org, err := c.service.CreateOrganization(c.toServiceOrganizationCreateRequest(modelReq), userID)
	if err != nil {
		problemErrors.HandleInternalError(ctx, "Failed to create organization", err)
		return
	}

	response := c.convertToOrganizationResponse(org)
	ctx.JSON(http.StatusCreated, response)
}

// GetOrganization obtém uma organização por ID
// @Summary Obter organização por ID
// @Description Obter uma organização específica por seu ID
// @Tags organizations
// @Accept json
// @Produce json
// @Param orgid path string true "ID da Organização"
// @Success 200 {object} controllers.OrganizationResponse "Informações da organização"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /organizations/{orgid} [get]
func (c *OrganizationController) GetOrganization(ctx *gin.Context) {
	id, err := c.getOrganizationIDFromPath(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	org, err := c.service.GetOrganization(id)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organização não encontrada")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to get organization", err)
		return
	}

	response := c.convertToOrganizationResponse(org)
	ctx.JSON(http.StatusOK, response)
}

// GetUserOrganizations obtém organizações criadas pelo usuário autenticado
// @Summary Obter organizações do usuário
// @Description Obter todas as organizações criadas pelo usuário autenticado
// @Tags organizations
// @Accept json
// @Produce json
// @Param page query int false "Número da página" default(1)
// @Param limit query int false "Itens por página" default(50)
// @Param sortBy query string false "Ordenar por campo" default("created_at")
// @Param sortOrder query string false "Ordem de classificação (asc/desc)" default("desc")
// @Success 200 {object} controllers.OrganizationListResponse "Lista das organizações do usuário"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /organizations/my [get]
func (c *OrganizationController) GetUserOrganizations(ctx *gin.Context) {
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	// Analisar parâmetros de paginação
	page, limit, sortBy, sortOrder := c.parsePaginationParams(ctx)
	offset := (page - 1) * limit

	orgs, total, err := c.service.GetUserOrganizations(userID, limit, offset, sortBy, sortOrder)
	if err != nil {
		problemErrors.HandleInternalError(ctx, "Failed to get user organizations", err)
		return
	}

	response := OrganizationListResponse{
		Data:  c.convertToOrganizationResponseList(orgs),
		Total: total,
		Limit: limit,
		Page:  page,
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateOrganization atualiza uma organização
// @Summary Atualizar organização
// @Description Atualizar uma organização (somente administradores)
// @Tags organizations
// @Accept json
// @Produce json
// @Param orgid path string true "ID da Organização"
// @Param organization body controllers.OrganizationUpdateRequest true "Dados de atualização"
// @Success 200 {object} controllers.OrganizationResponse "Organização atualizada com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /organizations/{orgid} [put]
func (c *OrganizationController) UpdateOrganization(ctx *gin.Context) {
	id, err := c.getOrganizationIDFromPath(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	var req OrganizationUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	// Convert to service format
	modelReq := &ServiceOrganizationUpdateRequest{}
	if req.Name != nil {
		modelReq.Name = req.Name
	}
	if req.Description != nil {
		modelReq.Description = req.Description
	}

	org, err := c.service.UpdateOrganization(id, c.toServiceOrganizationUpdateRequest(modelReq), userID)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organização não encontrada")
			return
		}
		if err.Error() == "insufficient permissions: only admins can update organization" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to update organization", err)
		return
	}

	response := c.convertToOrganizationResponse(org)
	ctx.JSON(http.StatusOK, response)
}

// DeleteOrganization exclui uma organização
// @Summary Excluir organização
// @Description Excluir uma organização (somente criador)
// @Tags organizations
// @Accept json
// @Produce json
// @Param orgid path string true "ID da Organização"
// @Success 204 "Organização excluída com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /organizations/{orgid} [delete]
func (c *OrganizationController) DeleteOrganization(ctx *gin.Context) {
	id, err := c.getOrganizationIDFromPath(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	err = c.service.DeleteOrganization(id, userID)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organização não encontrada")
			return
		}
		if err.Error() == "insufficient permissions: only the creator can delete organization" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to delete organization", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetOrganizationMembers obtém membros de uma organização
// @Summary Obter membros da organização
// @Description Obter todos os membros de uma organização
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param page query int false "Número da página" default(1)
// @Param limit query int false "Itens por página" default(50)
// @Param sortBy query string false "Ordenar por campo" default("joined_at")
// @Param sortOrder query string false "Ordem de classificação (asc/desc)" default("desc")
// @Success 200 {object} controllers.OrganizationMemberListResponse "Lista dos membros da organização"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /members [get]
func (c *OrganizationController) GetOrganizationMembers(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	// Analisar parâmetros de paginação
	page, limit, sortBy, sortOrder := c.parsePaginationParams(ctx)
	offset := (page - 1) * limit

	members, total, err := c.service.GetOrganizationMembers(orgID, userID, limit, offset, sortBy, sortOrder)
	if err != nil {
		if err.Error() == "insufficient permissions: user is not a member of this organization" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to get organization members", err)
		return
	}

	response := OrganizationMemberListResponse{
		Data:  c.convertToOrganizationMemberResponseList(members),
		Total: total,
		Limit: limit,
		Page:  page,
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateMemberRole atualiza a função de um membro
// @Summary Atualizar função do membro
// @Description Atualizar a função de um membro na organização (somente administradores)
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param userId path string true "ID do Usuário"
// @Param member body controllers.OrganizationMemberUpdateRequest true "Dados de atualização de função"
// @Success 200 {object} controllers.OrganizationMemberResponse "Função do membro atualizada com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Membro ou organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /members/{userId} [put]
func (c *OrganizationController) UpdateMemberRole(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	memberUserIDStr := ctx.Param("userId")
	memberUserID, err := uuid.Parse(memberUserIDStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Formato de ID de usuário inválido")
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	var req OrganizationMemberUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	// Convert to service format
	modelReq := &ServiceOrganizationMemberUpdateRequest{
		Role: req.Role,
	}

	member, err := c.service.UpdateMemberRole(orgID, memberUserID, c.toServiceOrganizationMemberUpdateRequest(modelReq), userID)
	if err != nil {
		if err.Error() == "organization not found" || err.Error() == "member not found" {
			problemErrors.HandleNotFoundError(ctx, "Recurso não encontrado")
			return
		}
		if err.Error() == "insufficient permissions: only admins can update member roles" ||
			err.Error() == "cannot change role of organization creator" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to update member role", err)
		return
	}

	response := c.convertToOrganizationMemberResponse(member)
	ctx.JSON(http.StatusOK, response)
}

// RemoveMember remove um membro de uma organização
// @Summary Remover membro
// @Description Remover um membro da organização
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param userId path string true "ID do Usuário"
// @Success 204 "Membro removido com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Membro ou organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /members/{userId} [delete]
func (c *OrganizationController) RemoveMember(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	memberUserIDStr := ctx.Param("userId")
	memberUserID, err := uuid.Parse(memberUserIDStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Formato de ID de usuário inválido")
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	err = c.service.RemoveMember(orgID, memberUserID, userID)
	if err != nil {
		if err.Error() == "organization not found" || err.Error() == "member not found" {
			problemErrors.HandleNotFoundError(ctx, "Recurso não encontrado")
			return
		}
		if err.Error() == "cannot remove organization creator" ||
			err.Error() == "insufficient permissions: only admins can remove other members" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to remove member", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// InviteUser envia um convite para ingressar em uma organização
// @Summary Convidar usuário para organização
// @Description Enviar um convite para um usuário ingressar na organização (somente administradores)
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param invitation body controllers.OrganizationInviteRequest true "Dados do convite"
// @Success 201 {object} controllers.OrganizationInviteResponse "Convite enviado com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /invites [post]
func (c *OrganizationController) InviteUser(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	var req OrganizationInviteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	// Convert to service format
	modelReq := &ServiceOrganizationInviteRequest{
		Email: req.Email,
		Role:  req.Role,
	}

	invite, err := c.service.InviteUser(orgID, c.toServiceOrganizationInviteRequest(modelReq), userID)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organização não encontrada")
			return
		}
		if err.Error() == "insufficient permissions: only admins can send invitations" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		if err.Error() == "user is already a member of this organization" ||
			err.Error() == "invitation already pending for this email" {
			problemErrors.HandleConflictError(ctx, err.Error())
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to send invitation", err)
		return
	}

	response := c.convertToOrganizationInviteResponse(invite)
	ctx.JSON(http.StatusCreated, response)
}

// GetOrganizationInvites recupera convites para uma organização
// @Summary Obter convites da organização
// @Description Obter todos os convites pendentes para a organização (somente administradores)
// @Tags organizations
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param page query int false "Número da página" default(1)
// @Param limit query int false "Itens por página" default(50)
// @Param sortBy query string false "Ordenar por campo" default("created_at")
// @Param sortOrder query string false "Ordem de classificação (asc/desc)" default("desc")
// @Success 200 {object} controllers.OrganizationInviteListResponse "Lista dos convites da organização"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /invites [get]
func (c *OrganizationController) GetOrganizationInvites(ctx *gin.Context) {
	orgID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	// Analisar parâmetros de paginação
	page, limit, sortBy, sortOrder := c.parsePaginationParams(ctx)
	offset := (page - 1) * limit

	invites, total, err := c.service.GetOrganizationInvites(orgID, userID, limit, offset, sortBy, sortOrder)
	if err != nil {
		if err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Organização não encontrada")
			return
		}
		if err.Error() == "insufficient permissions: only admins can view invitations" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to get organization invitations", err)
		return
	}

	response := OrganizationInviteListResponse{
		Data:  c.convertToOrganizationInviteResponseList(invites),
		Total: total,
		Limit: limit,
		Page:  page,
	}

	ctx.JSON(http.StatusOK, response)
}

// AcceptInvite aceita um convite de organização
// @Summary Aceitar convite da organização
// @Description Aceitar um convite para ingressar em uma organização usando o token de convite
// @Tags organizations
// @Accept json
// @Produce json
// @Param token path string true "Token de convite"
// @Success 200 {object} controllers.OrganizationMemberResponse "Convite aceito com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 404 {object} errors.NotFoundProblem "Convite não encontrado"
// @Failure 410 {object} errors.GoneProblem "Convite expirado ou não válido"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /invites/token/{token}/accept [post]
func (c *OrganizationController) AcceptInvite(ctx *gin.Context) {
	token := ctx.Param("token")
	if token == "" {
		problemErrors.HandleValidationError(ctx, "Token de convite obrigatório")
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	member, err := c.service.AcceptInvite(token, userID)
	if err != nil {
		if err.Error() == "invitation not found" {
			problemErrors.HandleNotFoundError(ctx, "Convite não encontrado")
			return
		}
		if err.Error() == "invitation is no longer valid" || err.Error() == "invitation has expired" {
			problemErrors.HandleGoneError(ctx, err.Error())
			return
		}
		if err.Error() == "user not found" {
			problemErrors.HandleNotFoundError(ctx, "Usuário não encontrado")
			return
		}
		if err.Error() == "invitation email does not match user email" ||
			err.Error() == "user is already a member of this organization" {
			problemErrors.HandleConflictError(ctx, err.Error())
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to accept invitation", err)
		return
	}

	response := c.convertToOrganizationMemberResponse(member)
	ctx.JSON(http.StatusOK, response)
}

// ValidateInvite valida um token de convite de organização sem exigir autenticação
// @Summary Validar convite da organização
// @Description Validar um token de convite e verificar se o usuário convidado existe
// @Tags organizations
// @Produce json
// @Param token path string true "Token de Convite"
// @Success 200 {object} controllers.OrganizationInviteResponse "Detalhes do convite válido"
// @Failure 404 {object} errors.NotFoundProblem "Convite não encontrado"
// @Failure 410 {object} errors.GoneProblem "Convite expirado ou não válido"
// @Router /invites/token/{token}/validate [get]
func (c *OrganizationController) ValidateInvite(ctx *gin.Context) {
	token := ctx.Param("token")
	if token == "" {
		problemErrors.HandleValidationError(ctx, "Token de convite obrigatório")
		return
	}

	result, err := c.service.ValidateInvite(token)
	if err != nil {
		if err.Error() == "invitation not found" {
			problemErrors.HandleNotFoundError(ctx, "Convite não encontrado")
			return
		}
		if err.Error() == "invitation is no longer valid" || err.Error() == "invitation has expired" {
			problemErrors.HandleGoneError(ctx, err.Error())
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to validate invitation", err)
		return
	}

	ctx.JSON(http.StatusOK, result)
}

// RevokeInvite revoga um convite de organização
// @Summary Revogar convite da organização
// @Description Revogar um convite para ingressar na organização (somente administradores)
// @Tags organizations
// @Accept json
// @Produce json
// @Param inviteId path string true "ID do Convite"
// @Success 204 "Convite revogado com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Convite não encontrado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /invites/{inviteId} [delete]
func (c *OrganizationController) RevokeInvite(ctx *gin.Context) {
	inviteIDStr := ctx.Param("inviteId")
	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Formato de ID de convite inválido")
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	err = c.service.RevokeInvite(inviteID, userID)
	if err != nil {
		if err.Error() == "invitation not found" || err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Recurso não encontrado")
			return
		}
		if err.Error() == "insufficient permissions: only admins can revoke invitations" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to revoke invitation", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetUserMemberships recupera todas as organizações das quais o usuário é membro
// @Summary Obter filiações do usuário
// @Description Obter todas as organizações das quais o usuário autenticado é membro
// @Tags organizations
// @Accept json
// @Produce json
// @Param page query int false "Número da página" default(1)
// @Param limit query int false "Itens por página" default(50)
// @Param sortBy query string false "Ordenar por campo" default("joined_at")
// @Param sortOrder query string false "Ordem de classificação (asc/desc)" default("desc")
// @Success 200 {object} controllers.OrganizationMemberListResponse "Lista das filiações do usuário"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /memberships [get]
func (c *OrganizationController) GetUserMemberships(ctx *gin.Context) {
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	// Analisar parâmetros de paginação
	page, limit, sortBy, sortOrder := c.parsePaginationParams(ctx)
	offset := (page - 1) * limit

	memberships, total, err := c.service.GetUserMemberships(userID, limit, offset, sortBy, sortOrder)
	if err != nil {
		problemErrors.HandleInternalError(ctx, "Failed to get user memberships", err)
		return
	}

	response := OrganizationMemberListResponse{
		Data:  c.convertToOrganizationMemberResponseList(memberships),
		Total: total,
		Limit: limit,
		Page:  page,
	}

	ctx.JSON(http.StatusOK, response)
}

// Métodos Auxiliares

// parsePaginationParams analisa parâmetros de paginação da query string
func (c *OrganizationController) parsePaginationParams(ctx *gin.Context) (page, limit int, sortBy, sortOrder string) {
	page = 1
	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit = 50
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	sortBy = ctx.DefaultQuery("sortBy", "created_at")
	sortOrder = ctx.DefaultQuery("sortOrder", "desc")

	return
}

// Métodos de Conversão

func (c *OrganizationController) convertToOrganizationResponse(org *models.Organization) *OrganizationResponse {
	response := &OrganizationResponse{
		ID:          org.ID,
		Name:        org.Name,
		Description: org.Description,
		CreatedBy:   org.CreatedBy,
		MemberCount: len(org.Members),
		InviteCount: len(org.Invites),
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
	}

	if org.Creator.ID != uuid.Nil {
		response.Creator = &UserResponse{
			ID:       org.Creator.ID,
			Username: org.Creator.Username,
			Name:     org.Creator.Name,
		}
	}

	return response
}

func (c *OrganizationController) convertToOrganizationResponseList(orgs []models.Organization) []OrganizationResponse {
	responses := make([]OrganizationResponse, len(orgs))
	for i, org := range orgs {
		responses[i] = *c.convertToOrganizationResponse(&org)
	}
	return responses
}

func (c *OrganizationController) convertToOrganizationMemberResponse(member *models.OrganizationMember) *OrganizationMemberResponse {
	response := &OrganizationMemberResponse{
		ID:             member.ID,
		OrganizationID: member.OrganizationID,
		UserID:         member.UserID,
		Role:           OrganizationRole(member.Role),
		JoinedAt:       member.JoinedAt,
		CreatedAt:      member.CreatedAt,
		UpdatedAt:      member.UpdatedAt,
	}

	if member.User.ID != uuid.Nil {
		response.User = &UserResponse{
			ID:       member.User.ID,
			Username: member.User.Username,
			Name:     member.User.Name,
		}
	}

	if member.Organization.ID != uuid.Nil {
		response.Organization = &OrganizationResponse{
			ID:          member.Organization.ID,
			Name:        member.Organization.Name,
			Description: member.Organization.Description,
			CreatedBy:   member.Organization.CreatedBy,
			CreatedAt:   member.Organization.CreatedAt,
			UpdatedAt:   member.Organization.UpdatedAt,
		}
	}

	return response
}

func (c *OrganizationController) convertToOrganizationMemberResponseList(members []models.OrganizationMember) []OrganizationMemberResponse {
	responses := make([]OrganizationMemberResponse, len(members))
	for i, member := range members {
		responses[i] = *c.convertToOrganizationMemberResponse(&member)
	}
	return responses
}

func (c *OrganizationController) convertToOrganizationInviteResponse(invite *models.OrganizationInvite) *OrganizationInviteResponse {
	response := &OrganizationInviteResponse{
		ID:             invite.ID,
		OrganizationID: invite.OrganizationID,
		Email:          invite.Email,
		Role:           OrganizationRole(invite.Role),
		InvitedBy:      invite.InvitedBy,
		Status:         InviteStatus(invite.Status),
		ExpiresAt:      invite.ExpiresAt,
		AcceptedAt:     invite.AcceptedAt,
		CreatedAt:      invite.CreatedAt,
		UpdatedAt:      invite.UpdatedAt,
	}

	if invite.Organization.ID != uuid.Nil {
		response.Organization = &OrganizationResponse{
			ID:          invite.Organization.ID,
			Name:        invite.Organization.Name,
			Description: invite.Organization.Description,
			CreatedBy:   invite.Organization.CreatedBy,
			CreatedAt:   invite.Organization.CreatedAt,
			UpdatedAt:   invite.Organization.UpdatedAt,
		}
	}

	if invite.Inviter.ID != uuid.Nil {
		response.Inviter = &UserResponse{
			ID:       invite.Inviter.ID,
			Username: invite.Inviter.Username,
			Name:     invite.Inviter.Name,
		}
	}

	return response
}

func (c *OrganizationController) convertToOrganizationInviteResponseList(invites []models.OrganizationInvite) []OrganizationInviteResponse {
	responses := make([]OrganizationInviteResponse, len(invites))
	for i, invite := range invites {
		responses[i] = *c.convertToOrganizationInviteResponse(&invite)
	}
	return responses
}

// ResendInvite reenvia um convite de organização
// @Summary Reenviar convite da organização
// @Description Reenviar um convite com um novo token e expiração estendida (somente administradores)
// @Tags organizations
// @Accept json
// @Produce json
// @Param inviteId path string true "ID do Convite"
// @Success 200 {object} controllers.OrganizationInviteResponse "Convite reenviado com sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 403 {object} errors.ForbiddenProblem "Permissões insuficientes"
// @Failure 404 {object} errors.NotFoundProblem "Convite não encontrado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /invites/{inviteId}/resend [post]
func (c *OrganizationController) ResendInvite(ctx *gin.Context) {
	inviteIDStr := ctx.Param("inviteId")
	inviteID, err := uuid.Parse(inviteIDStr)
	if err != nil {
		problemErrors.HandleValidationError(ctx, "Formato de ID de convite inválido")
		return
	}

	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		problemErrors.HandleValidationError(ctx, err.Error())
		return
	}

	invite, err := c.service.ResendInvite(inviteID, userID)
	if err != nil {
		if err.Error() == "invitation not found" || err.Error() == "organization not found" {
			problemErrors.HandleNotFoundError(ctx, "Recurso não encontrado")
			return
		}
		if err.Error() == "insufficient permissions: only admins can resend invitations" {
			problemErrors.HandleForbiddenError(ctx, "Permissões insuficientes")
			return
		}
		if err.Error() == "can only resend pending invitations" {
			problemErrors.HandleValidationError(ctx, "Can only resend pending invitations")
			return
		}
		problemErrors.HandleInternalError(ctx, "Failed to resend invitation", err)
		return
	}

	response := c.convertToOrganizationInviteResponse(invite)
	ctx.JSON(http.StatusOK, response)
}
