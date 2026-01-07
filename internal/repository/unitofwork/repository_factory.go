package unitofwork

import "context"

type RepositoryFactory interface {
	NewUnitOfWork(ctx context.Context) UnitOfWork
}
