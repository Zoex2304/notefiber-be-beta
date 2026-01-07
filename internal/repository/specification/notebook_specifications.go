package specification

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ByParentID struct {
	ParentID *uuid.UUID
}

func (s ByParentID) Apply(db *gorm.DB) *gorm.DB {
	if s.ParentID == nil {
		return db.Where("parent_id IS NULL")
	}
	return db.Where("parent_id = ?", s.ParentID)
}

// ByIDs is already in common_specifications, but we might want a domain specific alias or just use common?
// Using common is fine.
