package implementation

import (
	"context"
	"errors"
	"time"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/mapper"
	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/internal/repository/contract"
	"ai-notetaking-be/internal/repository/specification"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepositoryImpl struct {
	db     *gorm.DB
	mapper *mapper.UserMapper
}

func NewUserRepository(db *gorm.DB) contract.UserRepository {
	return &UserRepositoryImpl{
		db:     db,
		mapper: mapper.NewUserMapper(),
	}
}

func (r *UserRepositoryImpl) applySpecifications(db *gorm.DB, specs ...specification.Specification) *gorm.DB {
	for _, spec := range specs {
		db = spec.Apply(db)
	}
	return db
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *entity.User) error {
	modelUser := r.mapper.ToModel(user)
	if err := r.db.WithContext(ctx).Create(modelUser).Error; err != nil {
		return err
	}
	*user = *r.mapper.ToEntity(modelUser)
	return nil
}

func (r *UserRepositoryImpl) Update(ctx context.Context, user *entity.User) error {
	modelUser := r.mapper.ToModel(user)
	if err := r.db.WithContext(ctx).Save(modelUser).Error; err != nil {
		return err
	}
	*user = *r.mapper.ToEntity(modelUser)
	return nil
}

func (r *UserRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.User{}).Error
}

func (r *UserRepositoryImpl) DeleteUnscoped(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&model.User{}).Error
}

func (r *UserRepositoryImpl) FindOne(ctx context.Context, specs ...specification.Specification) (*entity.User, error) {
	var modelUser model.User
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)

	if err := query.First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return r.mapper.ToEntity(&modelUser), nil
}

func (r *UserRepositoryImpl) FindAll(ctx context.Context, specs ...specification.Specification) ([]*entity.User, error) {
	var modelUsers []*model.User
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)

	if err := query.Find(&modelUsers).Error; err != nil {
		return nil, err
	}

	return r.mapper.ToEntities(modelUsers), nil
}

func (r *UserRepositoryImpl) Count(ctx context.Context, specs ...specification.Specification) (int64, error) {
	var count int64
	query := r.applySpecifications(r.db.WithContext(ctx).Model(&model.User{}), specs...)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// FindOneUnscoped finds a user including soft-deleted ones (ignores deleted_at filter)
func (r *UserRepositoryImpl) FindOneUnscoped(ctx context.Context, specs ...specification.Specification) (*entity.User, error) {
	var modelUser model.User
	query := r.applySpecifications(r.db.WithContext(ctx).Unscoped(), specs...)

	if err := query.First(&modelUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return r.mapper.ToEntity(&modelUser), nil
}

// Restore reactivates a soft-deleted user by clearing deleted_at
func (r *UserRepositoryImpl) Restore(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Unscoped().Model(&model.User{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": nil,
			"status":     "active",
		}).Error
}

// Token Implementations

func (r *UserRepositoryImpl) CreatePasswordResetToken(ctx context.Context, token *entity.PasswordResetToken) error {
	m := r.mapper.PasswordResetTokenToModel(token)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

func (r *UserRepositoryImpl) FindPasswordResetToken(ctx context.Context, specs ...specification.Specification) (*entity.PasswordResetToken, error) {
	var m model.PasswordResetToken
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.PasswordResetTokenToEntity(&m), nil
}

func (r *UserRepositoryImpl) CreateEmailVerificationToken(ctx context.Context, token *entity.EmailVerificationToken) error {
	m := r.mapper.EmailVerificationTokenToModel(token)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

func (r *UserRepositoryImpl) FindEmailVerificationToken(ctx context.Context, specs ...specification.Specification) (*entity.EmailVerificationToken, error) {
	var m model.EmailVerificationToken
	query := r.applySpecifications(r.db.WithContext(ctx), specs...)
	if err := query.First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return r.mapper.EmailVerificationTokenToEntity(&m), nil
}

func (r *UserRepositoryImpl) CreateRefreshToken(ctx context.Context, token *entity.UserRefreshToken) error {
	m := r.mapper.UserRefreshTokenToModel(token)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return err
	}
	return nil
}

// Extended Implementation

func (r *UserRepositoryImpl) MarkTokenUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.PasswordResetToken{}).Where("id = ?", id).Update("used", true).Error
}

func (r *UserRepositoryImpl) DeleteEmailVerificationToken(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.EmailVerificationToken{}, id).Error
}

func (r *UserRepositoryImpl) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Model(&model.UserRefreshToken{}).Where("token_hash = ?", tokenHash).Update("revoked", true).Error
}

func (r *UserRepositoryImpl) GetByIdWithAvatar(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	var result struct {
		model.User
		AvatarUrlResolved *string `gorm:"column:avatar_url_resolved"`
	}

	err := r.db.WithContext(ctx).Table("users").
		Select("users.*, COALESCE(users.avatar_url, user_providers.avatar_url) as avatar_url_resolved").
		Joins("LEFT JOIN user_providers ON users.id = user_providers.user_id").
		Where("users.id = ?", id).
		Order("user_providers.created_at DESC").
		Limit(1).
		Scan(&result).Error

	if err != nil {
		// Scan with Limit 1 might not behave like First (returning valid record or error).
		// If ID not found, result fields will be zero.
		if result.Id == uuid.Nil {
			return nil, nil
		}
		return nil, err
	}

	if result.Id == uuid.Nil {
		return nil, nil
	}

	user := r.mapper.ToEntity(&result.User)
	if result.AvatarUrlResolved != nil {
		user.AvatarURL = result.AvatarUrlResolved
	}

	return user, nil
}

func (r *UserRepositoryImpl) ActivateUser(ctx context.Context, userId uuid.UUID) error {
	now := time.Now()
	// Using map for updates
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userId).
		Updates(map[string]interface{}{
			"status":            "active",
			"email_verified":    true,
			"email_verified_at": now,
		}).Error
}

func (r *UserRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

func (r *UserRepositoryImpl) UpdateAvatar(ctx context.Context, userId uuid.UUID, avatarURL string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userId).Update("avatar_url", avatarURL).Error
}

func (r *UserRepositoryImpl) UpdatePassword(ctx context.Context, userId uuid.UUID, hash string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userId).Update("password_hash", hash).Error
}

func (r *UserRepositoryImpl) SaveUserProvider(ctx context.Context, provider *entity.UserProvider) error {
	m := r.mapper.UserProviderToModel(provider)
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO user_providers (id, user_id, provider_name, provider_user_id, avatar_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (provider_name, provider_user_id) 
		DO UPDATE SET avatar_url = EXCLUDED.avatar_url
	`, m.Id, m.UserId, m.ProviderName, m.ProviderUserId, m.AvatarURL, m.CreatedAt).Error
}

func (r *UserRepositoryImpl) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*entity.User, error) {
	var models []*model.User
	pattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("email ILIKE ? OR full_name ILIKE ?", pattern, pattern).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, err
	}
	return r.mapper.ToEntities(models), nil
}

func (r *UserRepositoryImpl) GetUserGrowth(ctx context.Context) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT to_char(created_at, 'YYYY-MM-DD') as date, COUNT(*) as count 
		FROM users 
		WHERE created_at > NOW() - INTERVAL '30 days' 
		GROUP BY date 
		ORDER BY date ASC
	`).Scan(&results).Error
	return results, err
}

func (r *UserRepositoryImpl) CountByStatus(ctx context.Context, status string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", status).Count(&count).Error
	return int(count), err
}
