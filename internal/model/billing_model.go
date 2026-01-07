package model

import (
	"time"

	"github.com/google/uuid"
)

type BillingAddress struct {
	Id           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserId       uuid.UUID `gorm:"type:uuid;not null;index"`
	FirstName    string    `gorm:"type:varchar(255);not null"`
	LastName     string    `gorm:"type:varchar(255);not null"`
	Email        string    `gorm:"type:varchar(255);not null"`
	Phone        string    `gorm:"type:varchar(50)"`
	AddressLine1 string    `gorm:"type:varchar(255);not null"`
	AddressLine2 string    `gorm:"type:varchar(255)"`
	City         string    `gorm:"type:varchar(255);not null"`
	State        string    `gorm:"type:varchar(255);not null"`
	PostalCode   string    `gorm:"type:varchar(20);not null"`
	Country      string    `gorm:"type:varchar(255);not null"`
	IsDefault    bool      `gorm:"default:false"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (BillingAddress) TableName() string {
	return "billing_addresses"
}
