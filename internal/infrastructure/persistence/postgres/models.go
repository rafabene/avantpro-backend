package postgres

// UserModel é o model GORM para usuários
type UserModel struct {
	ID           string  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email        string  `gorm:"type:varchar(255);uniqueIndex;not null"`
	Name         string  `gorm:"type:varchar(500);not null"`
	PasswordHash string  `gorm:"type:varchar(255);not null"`
	Role         string  `gorm:"type:varchar(50);not null;index"`
	AvatarURL    *string `gorm:"type:varchar(500)"`
	CreatedAt    int64   `gorm:"autoCreateTime;index"`
	UpdatedAt    int64   `gorm:"autoUpdateTime"`
	DeletedAt    *int64  `gorm:"index"` // Soft delete
}

func (UserModel) TableName() string {
	return "users"
}
