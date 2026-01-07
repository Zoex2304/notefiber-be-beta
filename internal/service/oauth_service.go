// FILE: internal/service/oauth_service.go
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type IOAuthService interface {
	GetLoginURL(provider string) (string, error)
	HandleCallback(ctx context.Context, provider string, code string) (*dto.LoginResponse, error)
}

type oauthService struct {
	uowFactory unitofwork.RepositoryFactory
	googleConf *oauth2.Config
}

func NewOAuthService(uowFactory unitofwork.RepositoryFactory) IOAuthService {
	// Initialize Google OAuth Config
	conf := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	log.Printf("[OAuth Service] Initialized with:")
	log.Printf("  - Client ID: %s", conf.ClientID[:10]+"...")
	log.Printf("  - Redirect URL: %s", conf.RedirectURL)

	return &oauthService{
		uowFactory: uowFactory,
		googleConf: conf,
	}
}

func (s *oauthService) GetLoginURL(provider string) (string, error) {
	if provider != "google" {
		log.Printf("[OAuth Service] ERROR - Unsupported provider: %s", provider)
		return "", errors.New("unsupported provider")
	}

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)

	url := s.googleConf.AuthCodeURL(state)
	log.Printf("[OAuth Service] Generated login URL with state: %s", state)

	return url, nil
}

func (s *oauthService) HandleCallback(ctx context.Context, provider string, code string) (*dto.LoginResponse, error) {
	if provider != "google" {
		log.Printf("[OAuth Service] ERROR - Unsupported provider: %s", provider)
		return nil, errors.New("unsupported provider")
	}

	log.Printf("[OAuth Service] Starting callback handling...")

	// Exchange code for token
	token, err := s.googleConf.Exchange(ctx, code)
	if err != nil {
		log.Printf("[OAuth Service] ERROR - Code exchange failed: %v", err)
		return nil, fmt.Errorf("code exchange failed: %v", err)
	}
	log.Printf("[OAuth Service] ✅ Successfully exchanged code for access token")

	// Get user info from Google
	userInfoURL := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken
	log.Printf("[OAuth Service] Fetching user info from Google...")

	resp, err := http.Get(userInfoURL)
	if err != nil {
		log.Printf("[OAuth Service] ERROR - Failed getting user info: %v", err)
		return nil, fmt.Errorf("failed getting user info: %v", err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[OAuth Service] ERROR - Failed reading response: %v", err)
		return nil, fmt.Errorf("failed reading response: %v", err)
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}

	if err := json.Unmarshal(content, &googleUser); err != nil {
		log.Printf("[OAuth Service] ERROR - Failed to parse user info: %v", err)
		return nil, err
	}

	log.Printf("[OAuth Service] ✅ Received user info from Google:")
	log.Printf("  - Google ID: %s", googleUser.ID)
	log.Printf("  - Email: %s", googleUser.Email)
	log.Printf("  - Name: %s", googleUser.Name)
	log.Printf("  - Email Verified: %v", googleUser.VerifiedEmail)
	log.Printf("  - Picture: %s", googleUser.Picture)

	uow := s.uowFactory.NewUnitOfWork(ctx)

	// Check if user exists (including soft-deleted users to handle re-registration)
	log.Printf("[OAuth Service] Checking if user exists with email: %s", googleUser.Email)

	// First check for active users
	user, err := uow.UserRepository().FindOne(ctx, specification.ByEmail{Email: googleUser.Email})
	if err != nil {
		log.Printf("[OAuth Service] ERROR - Database query failed: %v", err)
		return nil, err
	}

	// If no active user found, check for soft-deleted users
	if user == nil {
		log.Printf("[OAuth Service] No active user found, checking for soft-deleted users...")
		user, err = uow.UserRepository().FindOneUnscoped(ctx, specification.ByEmail{Email: googleUser.Email})
		if err != nil {
			log.Printf("[OAuth Service] ERROR - Unscoped query failed: %v", err)
			return nil, err
		}

		if user != nil {
			// User was soft-deleted, reactivate them
			log.Printf("[OAuth Service] Found soft-deleted user with ID: %s. Reactivating...", user.Id)
			if err := uow.UserRepository().Restore(ctx, user.Id); err != nil {
				log.Printf("[OAuth Service] ERROR - Failed to restore user: %v", err)
				return nil, err
			}
			// Update the user object to reflect restored state
			user, _ = uow.UserRepository().FindOne(ctx, specification.ByEmail{Email: googleUser.Email})
			log.Printf("[OAuth Service] ✅ User reactivated successfully")
		}
	}

	// Create new user if doesn't exist at all
	if user == nil {
		log.Printf("[OAuth Service] User not found. Creating new user...")
		newUser := &entity.User{
			Id:            uuid.New(),
			Email:         googleUser.Email,
			FullName:      googleUser.Name,
			PasswordHash:  nil,
			Role:          entity.UserRoleUser,
			Status:        entity.UserStatusActive,
			EmailVerified: true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		// Use transaction even though single create for consistency/safety
		if err := uow.Begin(ctx); err != nil {
			return nil, err
		}

		if err := uow.UserRepository().Create(ctx, newUser); err != nil {
			uow.Rollback()
			log.Printf("[OAuth Service] ERROR - Failed to create user: %v", err)
			return nil, err
		}

		if err := uow.Commit(); err != nil {
			return nil, err
		}

		user = newUser
		log.Printf("[OAuth Service] ✅ New user created with ID: %s", user.Id)
	} else {
		log.Printf("[OAuth Service] ✅ Existing user found with ID: %s", user.Id)
	}

	// Sync Provider Info & Avatar
	log.Printf("[OAuth Service] Saving provider info...")
	userProvider := &entity.UserProvider{
		Id:             uuid.New(),
		UserId:         user.Id,
		ProviderName:   "google",
		ProviderUserId: googleUser.ID,
		AvatarURL:      googleUser.Picture,
		CreatedAt:      time.Now(),
	}

	// Assuming SaveUserProvider handles its own tx or is atomic enough, or we wrap in explicit tx?
	// Given it's a separate operation here, atomic single call is fine.
	if err := uow.UserRepository().SaveUserProvider(ctx, userProvider); err != nil {
		log.Printf("[OAuth Service] ERROR - Failed to save provider info: %v", err)
		return nil, fmt.Errorf("failed to save provider info: %v", err)
	}
	log.Printf("[OAuth Service] ✅ Provider info saved")

	// Generate JWT Token
	log.Printf("[OAuth Service] Generating JWT token...")
	claims := jwt.MapClaims{
		"user_id": user.Id.String(),
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	signedToken, err := jwtToken.SignedString([]byte(secret))
	if err != nil {
		log.Printf("[OAuth Service] ERROR - Failed to sign JWT: %v", err)
		return nil, err
	}
	log.Printf("[OAuth Service] ✅ JWT token generated successfully")

	loginResponse := &dto.LoginResponse{
		AccessToken: signedToken,
		User: dto.UserDTO{
			Id:       user.Id,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     string(user.Role),
		},
	}

	log.Printf("[OAuth Service] ✅ Login response prepared for user: %s", user.Email)

	return loginResponse, nil
}
