package model

import (
	"time"

	"github.com/google/uuid"
)

type AiCreditTransaction struct {
	Id              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserId          uuid.UUID  `gorm:"type:uuid;not null;index"`
	TransactionType string     `gorm:"type:ai_credit_transaction_type;not null"`
	Amount          int        `gorm:"not null"`
	ServiceUsed     *string    `gorm:"type:text;index"`
	RelatedId       *uuid.UUID `gorm:"type:uuid"`
	Notes           *string    `gorm:"type:text"`
	CreatedAt       time.Time  `gorm:"default:now();not null"`
}

func (AiCreditTransaction) TableName() string {
	return "ai_credit_transactions"
}
