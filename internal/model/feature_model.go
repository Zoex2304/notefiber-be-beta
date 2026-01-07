// FILE: internal/model/feature_model.go
// GORM model for the features (master catalog) table
package model

import (
	"time"

	"github.com/google/uuid"
)

// Feature represents a feature in the master catalog
type Feature struct {
	Id          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Key         string    `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	Category    string    `gorm:"type:varchar(50)"` // ai, storage, support, export
	IsActive    bool      `gorm:"default:true"`
	SortOrder   int       `gorm:"default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (Feature) TableName() string {
	return "features"
}
