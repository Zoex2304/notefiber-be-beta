package specification

import (
	"gorm.io/gorm"

	"github.com/google/uuid"
)

type ByEmail struct {
	Email string
}

func (s ByEmail) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("email = ?", s.Email)
}

type UserOwnedBy struct {
	UserID uuid.UUID
}

func (s UserOwnedBy) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("user_id = ?", s.UserID)
}

type ActiveUsers struct{}

func (s ActiveUsers) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("status = ?", "active")
}

// Token Specs

type ByToken struct {
	Token string
}

func (s ByToken) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("token = ?", s.Token)
}

type ByTokenHash struct {
	Hash string
}

func (s ByTokenHash) Apply(db *gorm.DB) *gorm.DB {
	return db.Where("token_hash = ?", s.Hash)
}
