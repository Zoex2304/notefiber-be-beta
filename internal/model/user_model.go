package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	Id                                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email                             string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash                      *string   `gorm:"type:varchar(255)"`
	FullName                          string    `gorm:"type:varchar(255);not null"`
	Role                              string    `gorm:"type:varchar(50);not null;default:'user'"`
	Status                            string    `gorm:"type:varchar(50);not null;default:'pending'"`
	EmailVerified                     bool      `gorm:"default:false"`
	EmailVerifiedAt                   *time.Time
	AvatarURL                         *string `gorm:"type:text"`
	AiDailyUsage                      int     `gorm:"default:0"`
	AiDailyUsageLastReset             time.Time
	SemanticSearchDailyUsage          int `gorm:"default:0"`
	SemanticSearchDailyUsageLastReset time.Time
	CreatedAt                         time.Time      `gorm:"autoCreateTime"`
	UpdatedAt                         time.Time      `gorm:"autoUpdateTime"`
	DeletedAt                         gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}

type PasswordResetToken struct {
	Id        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserId    uuid.UUID `gorm:"type:uuid;not null;index"`
	Token     string    `gorm:"type:varchar(255);not null;index"`
	ExpiresAt time.Time `gorm:"not null"`
	Used      bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

type UserProvider struct {
	Id             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserId         uuid.UUID `gorm:"type:uuid;not null;index"`
	ProviderName   string    `gorm:"type:varchar(50);not null"`
	ProviderUserId string    `gorm:"type:varchar(255);not null"`
	AvatarURL      string    `gorm:"type:text"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (UserProvider) TableName() string {
	return "user_providers"
}

type EmailVerificationToken struct {
	Id        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserId    uuid.UUID `gorm:"type:uuid;not null;index"`
	Token     string    `gorm:"type:varchar(255);not null;index"`
	ExpiresAt time.Time `gorm:"not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}

type UserRefreshToken struct {
	Id        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserId    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"type:text;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"default:false"`
	IpAddress string    `gorm:"type:varchar(45)"`
	UserAgent string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (UserRefreshToken) TableName() string {
	return "user_refresh_tokens"
}
