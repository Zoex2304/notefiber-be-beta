package scope

import "gorm.io/gorm"

func OrderByCreatedDesc(db *gorm.DB) *gorm.DB {
	return db.Order("created_at DESC")
}

func OrderByCreatedAsc(db *gorm.DB) *gorm.DB {
	return db.Order("created_at ASC")
}
