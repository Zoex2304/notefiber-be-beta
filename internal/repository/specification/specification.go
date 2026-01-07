package specification

import "gorm.io/gorm"

// Specification defines the interface for query specifications
type Specification interface {
	Apply(db *gorm.DB) *gorm.DB
}
