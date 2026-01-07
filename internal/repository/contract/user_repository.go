package contract

import (
	"context"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteUnscoped(ctx context.Context, id uuid.UUID) error // Hard delete
	FindOne(ctx context.Context, specs ...specification.Specification) (*entity.User, error)
	FindOneUnscoped(ctx context.Context, specs ...specification.Specification) (*entity.User, error) // Includes soft-deleted
	FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.User, error)
	Count(ctx context.Context, specs ...specification.Specification) (int64, error)
	Restore(ctx context.Context, id uuid.UUID) error // Reactivate soft-deleted user

	// Token Management (Usually part of user repo or separate, but putting here for cohesion)
	CreatePasswordResetToken(ctx context.Context, token *entity.PasswordResetToken) error
	FindPasswordResetToken(ctx context.Context, specs ...specification.Specification) (*entity.PasswordResetToken, error)
	MarkTokenUsed(ctx context.Context, id uuid.UUID) error

	CreateEmailVerificationToken(ctx context.Context, token *entity.EmailVerificationToken) error
	FindEmailVerificationToken(ctx context.Context, specs ...specification.Specification) (*entity.EmailVerificationToken, error)
	DeleteEmailVerificationToken(ctx context.Context, id uuid.UUID) error

	CreateRefreshToken(ctx context.Context, token *entity.UserRefreshToken) error
	RevokeRefreshToken(ctx context.Context, tokenHash string) error

	// Business Specific
	GetByIdWithAvatar(ctx context.Context, id uuid.UUID) (*entity.User, error)
	ActivateUser(ctx context.Context, userId uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateAvatar(ctx context.Context, userId uuid.UUID, avatarURL string) error
	UpdatePassword(ctx context.Context, userId uuid.UUID, hash string) error

	// Provider
	SaveUserProvider(ctx context.Context, provider *entity.UserProvider) error

	// Queries/Stats
	SearchUsers(ctx context.Context, query string, limit, offset int) ([]*entity.User, error)
	GetUserGrowth(ctx context.Context) ([]map[string]interface{}, error)
	CountByStatus(ctx context.Context, status string) (int, error)
}
