package model

import (
	"time"

	"github.com/google/uuid"
)

type UserBillingAddress struct {
	Id           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserId       uuid.UUID `gorm:"type:uuid;not null;index"`
	FirstName    string    `gorm:"type:text;not null"`
	LastName     string    `gorm:"type:text;not null"`
	Email        string    `gorm:"type:text;not null"`
	Phone        *string   `gorm:"type:text"`
	AddressLine1 string    `gorm:"type:text;not null"`
	AddressLine2 *string   `gorm:"type:text"`
	City         string    `gorm:"type:text;not null"`
	State        string    `gorm:"type:text;not null"`
	PostalCode   string    `gorm:"type:text;not null"`
	Country      string    `gorm:"type:text;default:'Indonesia';not null"`
	IsDefault    bool      `gorm:"default:false"`
	CreatedAt    time.Time `gorm:"default:now();not null"`
	UpdatedAt    time.Time `gorm:"default:now();not null"`
}

func (UserBillingAddress) TableName() string {
	return "user_billing_addresses"
}
