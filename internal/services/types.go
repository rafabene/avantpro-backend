package services

// NotificationType representa o tipo de notificação
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
)

// NotificationEvent representa o tipo de evento que aciona uma notificação
type NotificationEvent string

const (
	NotificationEventMemberJoined       NotificationEvent = "member_joined"
	NotificationEventMemberLeft         NotificationEvent = "member_left"
	NotificationEventMemberRoleChanged  NotificationEvent = "member_role_changed"
	NotificationEventInvitationSent     NotificationEvent = "invitation_sent"
	NotificationEventInvitationAccepted NotificationEvent = "invitation_accepted"
	NotificationEventInvitationExpired  NotificationEvent = "invitation_expired"
	NotificationEventOrganizationUpdate NotificationEvent = "organization_update"
)

// GetDescription retorna uma descrição legível para um evento de notificação
func (event NotificationEvent) GetDescription() string {
	switch event {
	case NotificationEventMemberJoined:
		return "Quando um novo membro entrar na organização"
	case NotificationEventMemberLeft:
		return "Quando um membro sair da organização"
	case NotificationEventMemberRoleChanged:
		return "Quando o papel de um membro for alterado"
	case NotificationEventInvitationSent:
		return "Quando um convite for enviado"
	case NotificationEventInvitationAccepted:
		return "Quando um convite for aceito"
	case NotificationEventInvitationExpired:
		return "Quando um convite expirar"
	case NotificationEventOrganizationUpdate:
		return "Quando a organização for atualizada"
	default:
		return string(event)
	}
}

// IsValid verifica se o evento é válido
func (event NotificationEvent) IsValid() bool {
	switch event {
	case NotificationEventMemberJoined,
		NotificationEventMemberLeft,
		NotificationEventMemberRoleChanged,
		NotificationEventInvitationSent,
		NotificationEventInvitationAccepted,
		NotificationEventInvitationExpired,
		NotificationEventOrganizationUpdate:
		return true
	default:
		return false
	}
}

// OrganizationRole representa o papel de um usuário em uma organização
type OrganizationRole string

const (
	OrganizationRoleAdmin OrganizationRole = "admin"
	OrganizationRoleUser  OrganizationRole = "user"
)

// InviteStatus representa o status de um convite da organização
type InviteStatus string

const (
	InviteStatusPending  InviteStatus = "pending"
	InviteStatusAccepted InviteStatus = "accepted"
	InviteStatusExpired  InviteStatus = "expired"
	InviteStatusRevoked  InviteStatus = "revoked"
)

// NotificationPreferenceBulkItem representa um item de preferência único na atualização em lote
type NotificationPreferenceBulkItem struct {
	Event   NotificationEvent `json:"event" validate:"required" example:"member_joined"`
	Enabled bool              `json:"enabled" example:"true"`
}
