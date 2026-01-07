package router

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/unitofwork"
)

// SimpleNuanceResolver resolves nuances directly from database
type SimpleNuanceResolver struct{}

// NewNuanceResolver creates a new simple nuance resolver
func NewNuanceResolver() *SimpleNuanceResolver {
	return &SimpleNuanceResolver{}
}

// GetNuanceByKey loads a nuance from the database
func (r *SimpleNuanceResolver) GetNuanceByKey(ctx context.Context, uow unitofwork.UnitOfWork, key string) (*entity.AiNuance, error) {
	return uow.AiConfigRepository().FindNuanceByKey(ctx, key)
}
