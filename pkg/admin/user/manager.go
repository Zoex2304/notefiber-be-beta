package user

import (
	"context"
	"fmt"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	adminEvents "ai-notetaking-be/pkg/admin/events"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Manager handles user-related admin operations
type Manager struct {
	logger    logger.ILogger
	publisher adminEvents.Publisher
}

// NewManager creates a new user manager
func NewManager(logger logger.ILogger, publisher adminEvents.Publisher) *Manager {
	return &Manager{
		logger:    logger,
		publisher: publisher,
	}
}

// Create creates a new user with password hashing and emits event
func (m *Manager) Create(ctx context.Context, uow unitofwork.UnitOfWork, req dto.AdminCreateUserRequest) (*entity.User, error) {
	// 1. Check existing
	existing, _ := uow.UserRepository().FindOne(ctx, specification.ByEmail{Email: req.Email})
	if existing != nil {
		return nil, fmt.Errorf("email already exists")
	}

	// 2. Hash Password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashStr := string(hash)

	// 3. Create User
	now := time.Now()
	user := &entity.User{
		Id:            uuid.New(),
		Email:         req.Email,
		FullName:      req.FullName,
		PasswordHash:  &hashStr,
		Role:          entity.UserRole(req.Role),
		Status:        entity.UserStatusActive, // Auto activate if admin creates
		EmailVerified: true,                    // Auto verify
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := uow.UserRepository().Create(ctx, user); err != nil {
		return nil, err
	}

	// Emit USER_REGISTERED Event (Admin Created)
	m.publisher.PublishUserRegistered(ctx, user.Id, user.Email, user.FullName, "admin_panel")

	return user, nil
}

// Update updates user fields
func (m *Manager) Update(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID, req dto.AdminUpdateUserRequest) (*entity.User, error) {
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Update fields if present
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Role != "" {
		user.Role = entity.UserRole(req.Role)
	}
	if req.Status != "" {
		user.Status = entity.UserStatus(req.Status)
	}
	if req.Avatar != "" {
		val := req.Avatar
		user.AvatarURL = &val
	}
	if req.AiDailyLimitOverride != nil {
		user.AiDailyLimitOverride = req.AiDailyLimitOverride
	}

	if err := uow.UserRepository().Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Delete removes a user
func (m *Manager) Delete(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID) error {
	// Log action
	m.logger.Info("ADMIN", "Deleted User", map[string]interface{}{
		"userId": userId.String(),
	})

	return uow.UserRepository().Delete(ctx, userId)
}

// FindAll retrieves users with pagination and optional search
func (m *Manager) FindAll(ctx context.Context, uow unitofwork.UnitOfWork, page, limit int, search string) ([]*entity.User, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var users []*entity.User
	var err error

	if search != "" {
		users, err = uow.UserRepository().SearchUsers(ctx, search, limit, offset)
	} else {
		users, err = uow.UserRepository().FindAll(ctx,
			specification.Pagination{Limit: limit, Offset: offset},
			specification.OrderBy{Field: "created_at", Desc: true},
		)
	}

	return users, err
}

// FindOne retrieves a single user by ID
func (m *Manager) FindOne(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID) (*entity.User, error) {
	return uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
}

// UpdateStatus updates user status
func (m *Manager) UpdateStatus(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID, status string) error {
	m.logger.Info("ADMIN", "Updated user status", map[string]interface{}{
		"userId": userId.String(),
		"status": status,
		"admin":  "system",
	})
	return uow.UserRepository().UpdateStatus(ctx, userId, status)
}
