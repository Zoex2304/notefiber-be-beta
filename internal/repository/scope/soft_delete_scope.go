package scope

import "gorm.io/gorm"

// WithSoftDelete ensures soft deleted records are included (if strictly needed)
// Usually you'd use db.Unscoped()
func WithSoftDelete(db *gorm.DB) *gorm.DB {
	return db.Unscoped()
}

// ExcludeSoftDelete is effectively the default behavior but made explicit
func ExcludeSoftDelete(db *gorm.DB) *gorm.DB {
	return db.Where("deleted_at IS NULL")
}
