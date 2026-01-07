package specification

import "gorm.io/gorm"

// NoteSearchQuery filters notes by title or content explicitly
type NoteSearchQuery struct {
	Query string
}

func (s NoteSearchQuery) Apply(db *gorm.DB) *gorm.DB {
	pattern := "%" + s.Query + "%"
	// Search in Title OR Content
	// Using ILIKE for Postgres (case insensitive)
	return db.Where("title ILIKE ? OR content ILIKE ?", pattern, pattern)
}

// ByNotebookName filters notes by their notebook's name (case-insensitive)
type ByNotebookName struct {
	Name string
}

func (s ByNotebookName) Apply(db *gorm.DB) *gorm.DB {
	pattern := "%" + s.Name + "%"
	// Requires JOIN with notebooks table unless already joined
	// We'll assume the repository handles the join or we add it safely if missing?
	// Actually, GORM helps distinct joins. Let's add the JOIN here to be safe.
	return db.Joins("JOIN notebooks ON notebooks.id = notes.notebook_id").
		Where("notebooks.name ILIKE ?", pattern)
}

// ByNoteTitle filters notes by exact title match (case-insensitive)
type ByNoteTitle struct {
	Title string
}

func (s ByNoteTitle) Apply(db *gorm.DB) *gorm.DB {
	pattern := "%" + s.Title + "%"
	return db.Where("title ILIKE ?", pattern)
}
