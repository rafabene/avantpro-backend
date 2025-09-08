package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	problemErrors "github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// NotificationController gerencia requisições HTTP para operações de notificação
type NotificationController struct {
	notificationService services.NotificationService
}

// NewNotificationController cria uma nova instância do NotificationController
func NewNotificationController(notificationService services.NotificationService) *NotificationController {
	return &NotificationController{
		notificationService: notificationService,
	}
}

// getOrganizationIDFromHeader extrai e valida o cabeçalho Organization-ID
func (c *NotificationController) getOrganizationIDFromHeader(ctx *gin.Context) (*uuid.UUID, error) {
	orgIDHeader := ctx.GetHeader("Organization-ID")
	if orgIDHeader == "" {
		return nil, errors.New("cabeçalho Organization-ID é obrigatório")
	}

	orgID, err := uuid.Parse(orgIDHeader)
	if err != nil {
		return nil, errors.New("formato de Organization-ID inválido")
	}

	return &orgID, nil
}

// getUserIDFromContext extrai o ID do usuário do contexto do token JWT
func (c *NotificationController) getUserIDFromContext(ctx *gin.Context) (uuid.UUID, error) {
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

// GetUserNotifications obtém notificações paginadas para o usuário autenticado
// @Summary Obter notificações do usuário
// @Description Recuperar lista paginada de notificações para o usuário autenticado
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param page query int false "Número da página (padrão: 1)" default(1)
// @Param limit query int false "Número de itens por página (padrão: 10, máx: 100)" default(10)
// @Param sortBy query string false "Ordenar por campo (title, type, read, created_at, updated_at)" default(created_at)
// @Param sortOrder query string false "Ordem de classificação (asc, desc)" default(desc)
// @Security BearerAuth
// @Success 200 {object} controllers.NotificationListResponse "Resposta de sucesso com notificações e paginação"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notifications [get]
func (c *NotificationController) GetUserNotifications(ctx *gin.Context) {
	// Obter ID do usuário do cabeçalho
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID da organização do cabeçalho
	organizationID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Analisar parâmetros de paginação
	page, err := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(ctx.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	sortBy := ctx.DefaultQuery("sortBy", "created_at")
	sortOrder := ctx.DefaultQuery("sortOrder", "desc")

	// Calcular deslocamento
	offset := (page - 1) * limit

	// Obter notificações
	notifications, total, err := c.notificationService.GetUserNotifications(
		userID,
		organizationID,
		limit,
		offset,
		sortBy,
		sortOrder,
	)
	if err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Calcular informações de paginação
	totalPages := (int(total) + limit - 1) / limit

	ctx.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetUnreadNotifications obtém todas as notificações não lidas para o usuário autenticado
// @Summary Obter notificações não lidas
// @Description Recuperar todas as notificações não lidas para o usuário autenticado
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Security BearerAuth
// @Success 200 {object} []controllers.NotificationResponse "Resposta de sucesso com notificações não lidas"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notifications/unread [get]
func (c *NotificationController) GetUnreadNotifications(ctx *gin.Context) {
	// Obter ID do usuário do cabeçalho
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID da organização do cabeçalho
	organizationID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter notificações não lidas
	notifications, err := c.notificationService.GetUnreadNotifications(userID, organizationID)
	if err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

// GetUnreadCount obtém a contagem de notificações não lidas para o usuário autenticado
// @Summary Obter contagem de notificações não lidas
// @Description Recuperar a contagem de notificações não lidas para o usuário autenticado
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Security BearerAuth
// @Success 200 {object} controllers.UnreadCountResponse "Resposta de sucesso com contagem não lida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notifications/unread-count [get]
func (c *NotificationController) GetUnreadCount(ctx *gin.Context) {
	// Obter ID do usuário do cabeçalho
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter ID da organização do cabeçalho
	organizationID, err := c.getOrganizationIDFromHeader(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Obter contagem não lida
	count, err := c.notificationService.GetUnreadCount(userID, organizationID)
	if err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

// MarkAsRead marca uma notificação específica como lida
// @Summary Marcar notificação como lida
// @Description Marcar uma notificação específica como lida para o usuário autenticado
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param notifId path string true "ID da Notificação"
// @Security BearerAuth
// @Success 200 {object} controllers.MessageResponse "Mensagem de sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 404 {object} errors.NotFoundProblem "Notificação não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notifications/{notifId}/read [put]
func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	// Obter ID do usuário do cabeçalho
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Analisar ID da notificação
	notificationIDStr := ctx.Param("notifId")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		prob := problemErrors.BadRequestError("ID de notificação inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Marcar como lida
	if err := c.notificationService.MarkAsRead(notificationID, userID); err != nil {
		if err.Error() == "notification not found" {
			prob := problemErrors.NotFoundError("Notificação não encontrada", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		if err.Error() == "notification does not belong to user" {
			prob := problemErrors.ForbiddenError("Acesso negado", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, MessageResponse{
		Message: "Notificação marcada como lida com sucesso",
	})
}

// MarkAllAsRead marca todas as notificações como lidas para o usuário autenticado
// @Summary Marcar todas as notificações como lidas
// @Description Marcar todas as notificações como lidas para o usuário autenticado
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Security BearerAuth
// @Success 200 {object} controllers.MessageResponse "Mensagem de sucesso"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notifications/mark-all-read [put]
func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	// Obter ID do usuário do cabeçalho
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Marcar todas como lidas
	if err := c.notificationService.MarkAllAsRead(userID); err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, MessageResponse{
		Message: "Todas as notificações marcadas como lidas com sucesso",
	})
}

// DeleteNotification exclui uma notificação específica
// @Summary Excluir notificação
// @Description Excluir uma notificação específica para o usuário autenticado
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Param notifId path string true "ID da Notificação"
// @Security BearerAuth
// @Success 200 {object} controllers.MessageResponse "Mensagem de sucesso"
// @Failure 400 {object} errors.BadRequestProblem "Requisição inválida"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 404 {object} errors.NotFoundProblem "Notificação não encontrada"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notifications/{notifId} [delete]
func (c *NotificationController) DeleteNotification(ctx *gin.Context) {
	// Obter ID do usuário do cabeçalho
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Analisar ID da notificação
	notificationIDStr := ctx.Param("notifId")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		prob := problemErrors.BadRequestError("ID de notificação inválido: "+err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Excluir notificação
	if err := c.notificationService.DeleteNotification(notificationID, userID); err != nil {
		if err.Error() == "notification not found" {
			prob := problemErrors.NotFoundError("Notificação não encontrada", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		if err.Error() == "notification does not belong to user" {
			prob := problemErrors.ForbiddenError("Acesso negado", problemErrors.GetInstance(ctx))
			problemErrors.RespondWithProblem(ctx, prob)
			return
		}
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, MessageResponse{
		Message: "Notificação excluída com sucesso",
	})
}

// DeleteAllNotifications exclui todas as notificações para o usuário autenticado
// @Summary Excluir todas as notificações
// @Description Excluir todas as notificações para o usuário autenticado
// @Tags notifications
// @Accept json
// @Produce json
// @Param Organization-ID header string true "ID da Organização"
// @Security BearerAuth
// @Success 200 {object} controllers.MessageResponse "Mensagem de sucesso"
// @Failure 401 {object} errors.UnauthorizedProblem "Não autorizado"
// @Failure 500 {object} errors.InternalServerProblem "Erro interno do servidor"
// @Router /notifications [delete]
func (c *NotificationController) DeleteAllNotifications(ctx *gin.Context) {
	// Obter ID do usuário do cabeçalho
	userID, err := c.getUserIDFromContext(ctx)
	if err != nil {
		prob := problemErrors.ValidationError(err.Error(), problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	// Excluir todas as notificações
	if err := c.notificationService.DeleteAllNotifications(userID); err != nil {
		prob := problemErrors.InternalError(problemErrors.GetInstance(ctx))
		problemErrors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, MessageResponse{
		Message: "Todas as notificações excluídas com sucesso",
	})
}
