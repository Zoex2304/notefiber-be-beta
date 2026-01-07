// FILE: internal/service/auth_service.go
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/mailer"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"ai-notetaking-be/pkg/events"
	pktNats "ai-notetaking-be/pkg/nats"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type IAuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req *dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error)
	ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error
	VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) error
	Logout(ctx context.Context, refreshToken string) error
	LoginAdmin(ctx context.Context, req *dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error)
}

type authService struct {
	uowFactory     unitofwork.RepositoryFactory
	emailService   mailer.IEmailService
	eventPublisher *pktNats.Publisher
}

func NewAuthService(uowFactory unitofwork.RepositoryFactory, emailService mailer.IEmailService, eventPublisher *pktNats.Publisher) IAuthService {
	return &authService{
		uowFactory:     uowFactory,
		emailService:   emailService,
		eventPublisher: eventPublisher,
	}
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n), nil
}

func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	// 1. Check for existing user
	existing, _ := uow.UserRepository().FindOne(ctx, specification.ByEmail{Email: req.Email})
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// 2. Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashStr := string(hash)

	// 3. Create User Entity
	user := &entity.User{
		Id:            uuid.New(),
		Email:         req.Email,
		FullName:      req.FullName,
		PasswordHash:  &hashStr,
		Role:          entity.UserRoleUser,
		Status:        entity.UserStatusPending,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 4. Save to DB
	// Use transaction for user + token creation safety
	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	if err := uow.UserRepository().Create(ctx, user); err != nil {
		return nil, err
	}

	// 5. Generate and save OTP
	otpCode, err := generateOTP()
	if err != nil {
		return nil, err
	}

	verificationToken := &entity.EmailVerificationToken{
		Id:        uuid.New(),
		UserId:    user.Id,
		Token:     otpCode,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}

	if err := uow.UserRepository().CreateEmailVerificationToken(ctx, verificationToken); err != nil {
		return nil, err
	}

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	// Log to console for dev convenience
	fmt.Printf(">>> [DEBUG OTP] OTP for %s is: %s <<<\n", user.Email, otpCode)

	// SEND REAL EMAIL
	go func() {
		emailErr := s.emailService.SendOTP(user.Email, otpCode)
		if emailErr != nil {
			fmt.Printf("Error sending registration email: %v\n", emailErr)
		}
	}()

	return &dto.RegisterResponse{Id: user.Id, Email: user.Email}, nil
}

func (s *authService) VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	user, err := uow.UserRepository().FindOne(ctx, specification.ByEmail{Email: req.Email})
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	if user.Status == entity.UserStatusActive {
		return nil
	}

	tokenEntity, err := uow.UserRepository().FindEmailVerificationToken(ctx,
		specification.UserOwnedBy{UserID: user.Id},
		specification.ByToken{Token: req.Token},
	)
	if err != nil {
		return errors.New("invalid otp code")
	}
	if tokenEntity == nil {
		return errors.New("invalid otp code")
	}

	if time.Now().After(tokenEntity.ExpiresAt) {
		return errors.New("otp code expired")
	}

	// Activate
	if err := uow.Begin(ctx); err != nil {
		return err
	}
	defer uow.Rollback()

	if err := uow.UserRepository().ActivateUser(ctx, user.Id); err != nil {
		return err
	}

	_ = uow.UserRepository().DeleteEmailVerificationToken(ctx, tokenEntity.Id)

	return uow.Commit()
}

func (s *authService) Login(ctx context.Context, req *dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	// 1. Check if user exists
	user, err := uow.UserRepository().FindOne(ctx, specification.ByEmail{Email: req.Email})
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// 2. Check if user has a password (might be OAuth only)
	if user.PasswordHash == nil {
		return nil, errors.New("user registered via OAuth")
	}

	// 3. SECURITY CHECK: Check if email is verified
	if user.Status == entity.UserStatusPending || !user.EmailVerified {
		return nil, errors.New("email not verified. please check your inbox for the otp code")
	}

	// 4. Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 5. Check if user is blocked/suspended
	if user.Status == entity.UserStatusBlocked {
		return nil, errors.New("user account is blocked")
	}

	// 6. Generate JWT
	accessTokenExpiry := time.Hour * 24

	claims := jwt.MapClaims{
		"user_id": user.Id.String(),
		"role":    user.Role,
		"exp":     time.Now().Add(accessTokenExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret"
	}
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	var rawRefreshToken string

	// Hanya buat Refresh Token jika "Remember Me" dicentang
	if req.RememberMe {
		rawRefreshToken = uuid.New().String()

		// Hash token
		hasher := sha256.New()
		hasher.Write([]byte(rawRefreshToken))
		tokenHash := hex.EncodeToString(hasher.Sum(nil))

		refreshTokenEntity := &entity.UserRefreshToken{
			Id:        uuid.New(),
			UserId:    user.Id,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(time.Hour * 24 * 30),
			Revoked:   false,
			CreatedAt: time.Now(),
			IpAddress: ipAddress,
			UserAgent: userAgent,
		}

		err = uow.UserRepository().CreateRefreshToken(ctx, refreshTokenEntity)
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %v", err)
		}
	}

	// PUBLISH EVENT
	if s.eventPublisher != nil {
		event := events.BaseEvent{
			Type: "USER_LOGIN",
			Data: map[string]interface{}{
				"user_id": user.Id,
				"device":  userAgent, // Simple mapping
				"time":    time.Now().Format(time.RFC822),
			},
			OccurredAt: time.Now(),
		}
		if err := s.eventPublisher.Publish(ctx, event); err != nil {
			fmt.Printf("[WARN] Failed to publish USER_LOGIN event: %v\n", err)
		}
	}

	return &dto.LoginResponse{
		AccessToken:  signedToken,
		RefreshToken: rawRefreshToken,
		User: dto.UserDTO{
			Id:       user.Id,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     string(user.Role),
		},
	}, nil
}

func (s *authService) LoginAdmin(ctx context.Context, req *dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	// Reuse the core login logic by calling a shared private method or just copy-pasting for safety/separation.
	// Since the core logic is identical except for the Role check, we can call s.Login first, then check the role?
	// NO, s.Login might fail if we add specific user logic later.
	// Better to implement specific Admin logic to be safe and explicit.

	uow := s.uowFactory.NewUnitOfWork(ctx)

	// 1. Check if user exists
	user, err := uow.UserRepository().FindOne(ctx, specification.ByEmail{Email: req.Email})
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// 2. STRICT ROLE CHECK
	if user.Role != entity.UserRoleAdmin {
		return nil, errors.New("access denied: admins only")
	}

	// 3. Password Check
	if user.PasswordHash == nil {
		return nil, errors.New("user registered via OAuth")
	}
	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 4. Status Check
	if user.Status == entity.UserStatusBlocked {
		return nil, errors.New("admin account is blocked")
	}

	// 5. Generate Token
	accessTokenExpiry := time.Hour * 24 // Admin session might need different expiry? Keeping same for now.

	claims := jwt.MapClaims{
		"user_id": user.Id.String(),
		"role":    user.Role,
		"exp":     time.Now().Add(accessTokenExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret"
	}
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	// Admin might not need refresh token flow usually, but let's support it if 'remember me' is checked
	var rawRefreshToken string
	if req.RememberMe {
		rawRefreshToken = uuid.New().String()
		hasher := sha256.New()
		hasher.Write([]byte(rawRefreshToken))
		tokenHash := hex.EncodeToString(hasher.Sum(nil))

		refreshTokenEntity := &entity.UserRefreshToken{
			Id:        uuid.New(),
			UserId:    user.Id,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(time.Hour * 24 * 30),
			Revoked:   false,
			CreatedAt: time.Now(),
			IpAddress: ipAddress,
			UserAgent: userAgent,
		}

		err = uow.UserRepository().CreateRefreshToken(ctx, refreshTokenEntity)
		if err != nil {
			return nil, fmt.Errorf("failed to create admin session: %v", err)
		}
	}

	return &dto.LoginResponse{
		AccessToken:  signedToken,
		RefreshToken: rawRefreshToken,
		User: dto.UserDTO{
			Id:       user.Id,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     string(user.Role),
		},
	}, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	uow := s.uowFactory.NewUnitOfWork(ctx)

	hasher := sha256.New()
	hasher.Write([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	return uow.UserRepository().RevokeRefreshToken(ctx, tokenHash)
}

func (s *authService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	user, err := uow.UserRepository().FindOne(ctx, specification.ByEmail{Email: req.Email})
	if err != nil || user == nil {
		// Don't leak exists
		return nil
	}

	token := uuid.New().String()
	resetToken := &entity.PasswordResetToken{
		Id:        uuid.New(),
		UserId:    user.Id,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
		Used:      false,
	}

	err = uow.UserRepository().CreatePasswordResetToken(ctx, resetToken)
	if err != nil {
		return err
	}

	go func() {
		emailErr := s.emailService.SendResetToken(user.Email, token)
		if emailErr != nil {
			fmt.Printf("Error sending reset password email: %v\n", emailErr)
		}
	}()

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	// FindPasswordResetToken by token
	tokenEntity, err := uow.UserRepository().FindPasswordResetToken(ctx, specification.ByToken{Token: req.Token})
	if err != nil || tokenEntity == nil {
		return errors.New("invalid or expired token")
	}

	if tokenEntity.Used {
		return errors.New("this password reset link has already been used")
	}

	if time.Now().After(tokenEntity.ExpiresAt) {
		return errors.New("this password reset link has expired")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if err := uow.Begin(ctx); err != nil {
		return err
	}
	defer uow.Rollback()

	err = uow.UserRepository().UpdatePassword(ctx, tokenEntity.UserId, string(hash))
	if err != nil {
		return err
	}

	err = uow.UserRepository().MarkTokenUsed(ctx, tokenEntity.Id)
	if err != nil {
		return err
	}

	return uow.Commit()
}
