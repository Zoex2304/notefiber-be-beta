package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AiConfiguration stores AI behavior settings (key-value pairs)
type AiConfiguration struct {
	Id          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Key         string         `gorm:"type:varchar(100);uniqueIndex;not null"`
	Value       string         `gorm:"type:text;not null"`
	ValueType   string         `gorm:"type:varchar(20);not null;default:'string'"`
	Description string         `gorm:"type:text"`
	Category    string         `gorm:"type:varchar(50);not null;default:'general';index"`
	IsSecret    bool           `gorm:"default:false"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (AiConfiguration) TableName() string {
	return "ai_configurations"
}

// AiNuance stores reusable prompt templates for behavior modification
type AiNuance struct {
	Id            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Key           string         `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name          string         `gorm:"type:varchar(200);not null"`
	Description   string         `gorm:"type:text"`
	SystemPrompt  string         `gorm:"type:text;not null"`
	ModelOverride *string        `gorm:"type:varchar(100)"`
	IsActive      bool           `gorm:"default:true;index"`
	SortOrder     int            `gorm:"default:0"`
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (AiNuance) TableName() string {
	return "ai_nuances"
}
