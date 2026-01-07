package specification

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ByID filters by ID
type ByID struct {
	ID uuid.UUID
}

func (s ByID) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("id = ?", s.ID)
}

// ByIDs filters by a list of IDs
type ByIDs struct {
	IDs []uuid.UUID
}

func (s ByIDs) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("id IN ?", s.IDs)
}

// OrderBy applies ordering
type OrderBy struct {
	Field string
	Desc  bool
}

func (s OrderBy) Apply(db *gorm.DB) *gorm.DB {
	direction := "ASC"
	if s.Desc {
		direction = "DESC"
	}
	return db.Order(fmt.Sprintf("%s %s", s.Field, direction))
}

// NotDeleted filters out soft-deleted records (explicitly)
// Note: GORM handles soft delete automatically if DeletedAt is present,
// but this can be used to be explicit or if global scope is disabled.
type NotDeleted struct{}

func (s NotDeleted) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("deleted_at IS NULL")
}

// Pagination
type Pagination struct {
	Limit  int
	Offset int
}

func (s Pagination) Apply(db *gorm.DB) *gorm.DB {
	return db.Limit(s.Limit).Offset(s.Offset)
}

// FilterBy Generic Filter
type FilterBy struct {
	Field string
	Value interface{}
}

func (s FilterBy) Apply(db *gorm.DB) *gorm.DB {
	query := fmt.Sprintf("%s = ?", s.Field)
	return db.Where(query, s.Value)
}

func Filter(field string, value interface{}) Specification {
	return FilterBy{Field: field, Value: value}
}
