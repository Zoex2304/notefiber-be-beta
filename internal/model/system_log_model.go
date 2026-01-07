package model

import (
	"time"

	"github.com/google/uuid"
)

type SystemLog struct {
	Id        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Level     string    `gorm:"type:varchar(20);not null;index"`
	Module    *string   `gorm:"type:varchar(50)"`
	Message   string    `gorm:"type:text;not null"`
	Details   *string   `gorm:"type:jsonb"`
	CreatedAt time.Time `gorm:"default:now();not null;index"`
}

func (SystemLog) TableName() string {
	return "system_logs"
}
