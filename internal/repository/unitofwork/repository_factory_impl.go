package unitofwork

import (
	"context"

	"gorm.io/gorm"
)

type RepositoryFactoryImpl struct {
	db *gorm.DB
}

func NewRepositoryFactory(db *gorm.DB) RepositoryFactory {
	return &RepositoryFactoryImpl{
		db: db,
	}
}

func (f *RepositoryFactoryImpl) NewUnitOfWork(ctx context.Context) UnitOfWork {
	// Usually UoW is short lived per request.
	// We pass the global DB instance.
	// The context can be used when calling Begin(), or explicitly passed to repos.
	// Here we just init the UoW.
	return NewUnitOfWork(f.db)
}
