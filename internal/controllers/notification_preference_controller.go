package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	problemErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// NotificationPreferenceController gerencia requisições HTTP para operações de preferência de notificação
type NotificationPreferenceController struct {
	preferenceService services.NotificationPreferenceService
}

// Conversion functions
func (c *NotificationPreferenceController) toServiceNotificationPreferenceBulkUpdateRequest(req *ServiceNotificationPreferenceBulkUpdateRequest) *services.NotificationPreferenceBulkUpdateRequest {
	servicePrefs := make([]services.NotificationPreferenceBulkItem, len(req.Preferences))
	for i, pref := range req.Preferences {
		servicePrefs[i] = services.NotificationPreferenceBulkItem{
			Event:   services.NotificationEvent(pref.Event),
			Enabled: pref.Enabled,
		}
	}
	return &services.NotificationPreferenceBulkUpdateRequest{
		Preferences: servicePrefs,
	}
}

func (c *NotificationPreferenceController) toServiceNotificationPreferenceUpdateRequest(req *ServiceNotificationPreferenceUpdateRequest) *services.NotificationPreferenceUpdateRequest {
	return &services.NotificationPreferenceUpdateRequest{
		Enabled: req.Enabled,
	}
}

func (c *NotificationPreferenceController) toServiceTestNotificationRequest(req *ServiceTestNotificationRequest) *services.TestNotificationRequest {
	return &services.TestNotificationRequest{
		OrganizationID: req.OrganizationID,
		Type:           services.NotificationType(req.Type),
		Title:          req.Title,
		Message:        req.Message,
	}
}

// NewNotificationPreferenceController cria uma nova instância do NotificationPreferenceController
func NewNotificationPreferenceController(preferenceService services.NotificationPreferenceService) *NotificationPreferenceController {
	return &NotificationPreferenceController{
		preferenceService: preferenceService,
	}
}

// getOrganizationIDFromHeader extrai e valida o cabeçalho Organization-ID
func (c *NotificationPreferenceController) getOrganizationIDFromHeader(ctx *gin.Context) (uuid.UUID, error) {
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
func (c *NotificationPreferenceController) getUserIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
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

// GetOrganizationPreferences obtém preferências de notificação para uma organização
// @Summary Obter preferências de notificação da organização
// @Description Recuperar preferências de notificação para uma organização. Cria padrões se nenhum existir.
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Organization-ID header string true "ID da Organização"
// @Success 200 {object} controllers.NotificationPreferenceListResponse "Resposta de sucesso com preferências"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notification-preferences [get]
func (c *NotificationPreferenceController) GetOrganizationPreferences(ctx *gin.Context) {
	// Verificar se o usuário está autenticado (middleware JWT já valida isso)
	_, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.UnauthorizedError("Usuário não autenticado", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID da organização do cabeçalho
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter preferências
	preferences, err := c.preferenceService.GetOrganizationPreferences(organizationUUID)
	if err != nil {
		if err.Error() == "organization not found" {
			prob := problemErrors.NotFoundError("Organização não encontrada", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"preferences": preferences,
	})
}

// UpdateOrganizationPreferences atualiza preferências de notificação para uma organização
// @Summary Atualizar preferências de notificação da organização
// @Description Atualização em lote das preferências de notificação para uma organização
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param request body controllers.NotificationPreferenceBulkUpdateRequest true "Solicitação de atualização em lote"
// @Security BearerAuth
// @Success 200 {object} controllers.NotificationPreferenceListResponse "Resposta de sucesso com preferências atualizadas"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notification-preferences [put]
func (c *NotificationPreferenceController) UpdateOrganizationPreferences(ctx *gin.Context) {
	// Verificar se o usuário está autenticado (middleware JWT já valida isso)
	_, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.UnauthorizedError("Usuário não autenticado", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID da organização do cabeçalho
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Analisar corpo da requisição
	var req NotificationPreferenceBulkUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Formato JSON inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Convert to service format
	modelReq := &ServiceNotificationPreferenceBulkUpdateRequest{
		Preferences: make([]ServiceNotificationPreferenceBulkItem, len(req.Preferences)),
	}
	for i, pref := range req.Preferences {
		enabled := pref.Enabled
		modelReq.Preferences[i] = ServiceNotificationPreferenceBulkItem{
			Event:   pref.Event,
			Enabled: enabled,
		}
	}

	// Atualizar preferências
	preferences, err := c.preferenceService.UpdateOrganizationPreferences(organizationUUID, c.toServiceNotificationPreferenceBulkUpdateRequest(modelReq))
	if err != nil {
		if err.Error() == "organization not found" {
			prob := problemErrors.NotFoundError("Organização não encontrada", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.BadRequestError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"preferences": preferences,
		"message":     "Preferências atualizadas com sucesso",
	})
}

// UpdateSinglePreference atualiza uma única preferência de notificação
// @Summary Atualizar preferência de notificação única
// @Description Atualizar uma preferência de notificação específica para uma organização
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param event path string true "Tipo de evento de notificação"
// @Param request body controllers.NotificationPreferenceUpdateRequest true "Solicitação de atualização"
// @Security BearerAuth
// @Success 200 {object} controllers.NotificationPreferenceResponse "Resposta de sucesso com preferência atualizada"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notification-preferences/{event} [put]
func (c *NotificationPreferenceController) UpdateSinglePreference(ctx *gin.Context) {
	// Verificar se o usuário está autenticado (middleware JWT já valida isso)
	_, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.UnauthorizedError("Usuário não autenticado", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID da organização do cabeçalho
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Analisar parâmetro do evento
	eventStr := ctx.Param("event")
	event := NotificationEvent(eventStr)

	// Analisar corpo da requisição
	var req NotificationPreferenceUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Formato JSON inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Convert to service format
	modelReq := &ServiceNotificationPreferenceUpdateRequest{
		Enabled: req.Enabled,
	}

	// Atualizar preferência única
	preference, err := c.preferenceService.UpdateSinglePreference(organizationUUID, services.NotificationEvent(event), c.toServiceNotificationPreferenceUpdateRequest(modelReq))
	if err != nil {
		if err.Error() == "organization not found" {
			prob := problemErrors.NotFoundError("Organização não encontrada", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.BadRequestError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"preference": preference,
		"message":    "Preferência atualizada com sucesso",
	})
}

// ResetToDefaults redefine preferências da organização para valores padrão
// @Summary Redefinir preferências para padrões
// @Description Redefinir todas as preferências de notificação para uma organização para valores padrão
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Security BearerAuth
// @Success 200 {object} controllers.NotificationPreferenceListResponse "Resposta de sucesso com preferências padrão"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 404 {object} errors.NotFoundProblem "Organização não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notification-preferences/reset [post]
func (c *NotificationPreferenceController) ResetToDefaults(ctx *gin.Context) {
	// Verificar se o usuário está autenticado (middleware JWT já valida isso)
	_, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.UnauthorizedError("Usuário não autenticado", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID da organização do cabeçalho
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Redefinir para padrões
	preferences, err := c.preferenceService.ResetToDefaults(organizationUUID)
	if err != nil {
		if err.Error() == "organization not found" {
			prob := problemErrors.NotFoundError("Organização não encontrada", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"preferences": preferences,
		"message":     "Preferências redefinidas para padrões com sucesso",
	})
}

// GetAvailableEvents retorna todos os eventos de notificação disponíveis
// @Summary Obter eventos de notificação disponíveis
// @Description Recuperar todos os tipos de evento de notificação disponíveis com descrições
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} controllers.NotificationEventsResponse "Resposta de sucesso com eventos disponíveis"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Router /notification-preferences/events [get]
func (c *NotificationPreferenceController) GetAvailableEvents(ctx *gin.Context) {
	// Verificar se o usuário está autenticado (middleware JWT já valida isso)
	_, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.UnauthorizedError("Usuário não autenticado", problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter eventos disponíveis
	events := c.preferenceService.GetAvailableEvents()

	ctx.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// GenerateTestNotification cria uma notificação de teste
// @Summary Gerar notificação de teste
// @Description Gerar uma notificação de teste para o usuário autenticado verificar as configurações
// @Tags notification-preferences
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param request body controllers.TestNotificationRequest true "Solicitação de notificação de teste"
// @Security BearerAuth
// @Success 201 {object} controllers.TestNotificationResponse "Resposta de sucesso com notificação de teste"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notification-preferences/test [post]
func (c *NotificationPreferenceController) GenerateTestNotification(ctx *gin.Context) {
	// Obter ID do usuário do cabeçalho
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID da organização do cabeçalho
	organizationUUID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Analisar corpo da requisição
	var req TestNotificationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := problemErrors.BadRequestError("Formato JSON inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Definir ID da organização do parâmetro do caminho
	req.OrganizationID = organizationUUID

	// Convert to service format
	modelReq := &ServiceTestNotificationRequest{
		Title:          req.Title,
		Message:        req.Message,
		Type:           req.Type,
		OrganizationID: req.OrganizationID,
	}

	// Gerar notificação de teste
	notification, err := c.preferenceService.GenerateTestNotification(userID, c.toServiceTestNotificationRequest(modelReq))
	if err != nil {
		if err.Error() == "user not found" {
			prob := problemErrors.NotFoundError("Usuário não encontrado", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"notification": notification,
		"message":      "Notificação de teste criada com sucesso",
	})
}
