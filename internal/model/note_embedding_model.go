package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type NoteEmbedding struct {
	Id             uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Document       string          `gorm:"type:text"`
	EmbeddingValue pgvector.Vector `gorm:"type:vector(768)"` // Gemini text-embedding-004 uses 768 dimensions
	NoteId         uuid.UUID       `gorm:"type:uuid;not null;index"`
	ChunkIndex     int             `gorm:"default:0"` // 0-based index for ordering
	CreatedAt      time.Time       `gorm:"autoCreateTime"`
	UpdatedAt      time.Time       `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt  `gorm:"index"`
}

func (NoteEmbedding) TableName() string {
	return "note_embeddings"
}
