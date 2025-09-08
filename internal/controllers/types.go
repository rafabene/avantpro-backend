package controllers

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

// NotificationType representa o tipo de notificação
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeSuccess NotificationType = "success"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
)

// NotificationPreferenceBulkItem representa um item de preferência único na atualização em lote
type NotificationPreferenceBulkItem struct {
	Event   NotificationEvent `json:"event" validate:"required" example:"member_joined"`
	Enabled bool              `json:"enabled" example:"true"`
}
