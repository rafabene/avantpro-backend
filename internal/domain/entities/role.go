package entities

// Role representa o papel de um usuário no sistema
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

// Permission representa uma permissão específica
type Permission string

const (
	// User permissions
	PermissionUserRead   Permission = "users.read"
	PermissionUserWrite  Permission = "users.write"
	PermissionUserDelete Permission = "users.delete"

	// Subscription permissions
	PermissionSubscriptionRead  Permission = "subscriptions.read"
	PermissionSubscriptionWrite Permission = "subscriptions.write"

	// Payment permissions
	PermissionPaymentRead  Permission = "payments.read"
	PermissionPaymentWrite Permission = "payments.write"
)

// RolePermissions mapeia roles para suas permissões
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		PermissionUserRead,
		PermissionUserWrite,
		PermissionUserDelete,
		PermissionSubscriptionRead,
		PermissionSubscriptionWrite,
		PermissionPaymentRead,
		PermissionPaymentWrite,
	},
	RoleUser: {
		PermissionUserRead,
		PermissionSubscriptionRead,
		PermissionSubscriptionWrite,
	},
	RoleGuest: {
		PermissionUserRead,
	},
}

// GetPermissions retorna permissões de um role
func (r Role) GetPermissions() []Permission {
	return RolePermissions[r]
}

// HasPermission verifica se role tem permissão
func (r Role) HasPermission(permission Permission) bool {
	permissions := RolePermissions[r]
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}
