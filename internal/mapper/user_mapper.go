package mapper

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"
)

type UserMapper struct{}

func NewUserMapper() *UserMapper {
	return &UserMapper{}
}

func (m *UserMapper) ToEntity(u *model.User) *entity.User {
	if u == nil {
		return nil
	}
	return &entity.User{
		Id:              u.Id,
		Email:           u.Email,
		PasswordHash:    u.PasswordHash,
		FullName:        u.FullName,
		Role:            entity.UserRole(u.Role),
		Status:          entity.UserStatus(u.Status),
		EmailVerified:   u.EmailVerified,
		EmailVerifiedAt: u.EmailVerifiedAt,
		AvatarURL:       u.AvatarURL,
		AiDailyUsage:    u.AiDailyUsage,

		AiDailyUsageLastReset:             u.AiDailyUsageLastReset,
		SemanticSearchDailyUsage:          u.SemanticSearchDailyUsage,
		SemanticSearchDailyUsageLastReset: u.SemanticSearchDailyUsageLastReset,
		CreatedAt:                         u.CreatedAt,
		UpdatedAt:                         u.UpdatedAt,
	}
}

func (m *UserMapper) ToModel(u *entity.User) *model.User {
	if u == nil {
		return nil
	}
	return &model.User{
		Id:              u.Id,
		Email:           u.Email,
		PasswordHash:    u.PasswordHash,
		FullName:        u.FullName,
		Role:            string(u.Role),
		Status:          string(u.Status),
		EmailVerified:   u.EmailVerified,
		EmailVerifiedAt: u.EmailVerifiedAt,
		AvatarURL:       u.AvatarURL,
		AiDailyUsage:    u.AiDailyUsage,

		AiDailyUsageLastReset:             u.AiDailyUsageLastReset,
		SemanticSearchDailyUsage:          u.SemanticSearchDailyUsage,
		SemanticSearchDailyUsageLastReset: u.SemanticSearchDailyUsageLastReset,
		CreatedAt:                         u.CreatedAt,
		UpdatedAt:                         u.UpdatedAt,
	}
}

func (m *UserMapper) ToEntities(users []*model.User) []*entity.User {
	entities := make([]*entity.User, len(users))
	for i, u := range users {
		entities[i] = m.ToEntity(u)
	}
	return entities
}

func (m *UserMapper) ToModels(users []*entity.User) []*model.User {
	models := make([]*model.User, len(users))
	for i, u := range users {
		models[i] = m.ToModel(u)
	}
	return models
}

// Token Mappers

func (m *UserMapper) PasswordResetTokenToEntity(t *model.PasswordResetToken) *entity.PasswordResetToken {
	if t == nil {
		return nil
	}
	return &entity.PasswordResetToken{
		Id:        t.Id,
		UserId:    t.UserId,
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
		Used:      t.Used,
		CreatedAt: t.CreatedAt,
	}
}

func (m *UserMapper) PasswordResetTokenToModel(t *entity.PasswordResetToken) *model.PasswordResetToken {
	if t == nil {
		return nil
	}
	return &model.PasswordResetToken{
		Id:        t.Id,
		UserId:    t.UserId,
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
		Used:      t.Used,
		CreatedAt: t.CreatedAt,
	}
}

func (m *UserMapper) UserProviderToEntity(p *model.UserProvider) *entity.UserProvider {
	if p == nil {
		return nil
	}
	return &entity.UserProvider{
		Id:             p.Id,
		UserId:         p.UserId,
		ProviderName:   p.ProviderName,
		ProviderUserId: p.ProviderUserId,
		AvatarURL:      p.AvatarURL,
		CreatedAt:      p.CreatedAt,
	}
}

func (m *UserMapper) UserProviderToModel(p *entity.UserProvider) *model.UserProvider {
	if p == nil {
		return nil
	}
	return &model.UserProvider{
		Id:             p.Id,
		UserId:         p.UserId,
		ProviderName:   p.ProviderName,
		ProviderUserId: p.ProviderUserId,
		AvatarURL:      p.AvatarURL,
		CreatedAt:      p.CreatedAt,
	}
}

func (m *UserMapper) EmailVerificationTokenToEntity(t *model.EmailVerificationToken) *entity.EmailVerificationToken {
	if t == nil {
		return nil
	}
	return &entity.EmailVerificationToken{
		Id:        t.Id,
		UserId:    t.UserId,
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}
}

func (m *UserMapper) EmailVerificationTokenToModel(t *entity.EmailVerificationToken) *model.EmailVerificationToken {
	if t == nil {
		return nil
	}
	return &model.EmailVerificationToken{
		Id:        t.Id,
		UserId:    t.UserId,
		Token:     t.Token,
		ExpiresAt: t.ExpiresAt,
		CreatedAt: t.CreatedAt,
	}
}

func (m *UserMapper) UserRefreshTokenToEntity(t *model.UserRefreshToken) *entity.UserRefreshToken {
	if t == nil {
		return nil
	}
	return &entity.UserRefreshToken{
		Id:        t.Id,
		UserId:    t.UserId,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		Revoked:   t.Revoked,
		IpAddress: t.IpAddress,
		UserAgent: t.UserAgent,
		CreatedAt: t.CreatedAt,
	}
}

func (m *UserMapper) UserRefreshTokenToModel(t *entity.UserRefreshToken) *model.UserRefreshToken {
	if t == nil {
		return nil
	}
	return &model.UserRefreshToken{
		Id:        t.Id,
		UserId:    t.UserId,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		Revoked:   t.Revoked,
		IpAddress: t.IpAddress,
		UserAgent: t.UserAgent,
		CreatedAt: t.CreatedAt,
	}
}
