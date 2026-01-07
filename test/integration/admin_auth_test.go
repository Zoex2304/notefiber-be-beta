package integration

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ai-notetaking-be/internal/bootstrap"
	"ai-notetaking-be/internal/config"
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/server"
	"ai-notetaking-be/pkg/database"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAdminAuth(t *testing.T) {
	// Setup
	// Load .env from root (2 levels up) because tests run in package dir
	if err := godotenv.Load("../../.env"); err != nil {
		t.Logf("Warning: Could not load ../../.env: %v", err)
	}
	cfg := config.Load()

	// Use correct DB init
	db, err := database.NewGormDBFromDSN(cfg.Database.Connection)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	container := bootstrap.NewContainer(db, cfg)
	srv := server.New(cfg, container)
	app := srv.GetApp()

	// Clean up test data helpers
	// uow removed

	// 1. Seed Admin User
	adminPass := "admin123"
	adminHash, _ := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	adminHashStr := string(adminHash)

	adminId := uuid.New()
	adminUser := entity.User{
		Id:            adminId,
		Email:         "testadmin@example.com",
		FullName:      "Test Admin",
		PasswordHash:  &adminHashStr,
		Role:          entity.UserRoleAdmin,
		Status:        entity.UserStatusActive,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 2. Seed Regular User
	userId := uuid.New()
	userUser := entity.User{
		Id:            userId,
		Email:         "testuser@example.com",
		FullName:      "Test User",
		PasswordHash:  &adminHashStr,
		Role:          entity.UserRoleUser,
		Status:        entity.UserStatusActive,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	db.Create(&adminUser)
	db.Create(&userUser)

	defer func() {
		// Cleanup
		db.Delete(&entity.User{}, adminId)
		db.Delete(&entity.User{}, userId)
	}()

	t.Run("Login as Admin success", func(t *testing.T) {
		reqBody := dto.LoginRequest{
			Email:    "testadmin@example.com",
			Password: "admin123",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req, -1)

		assert.Equal(t, 200, resp.StatusCode)

		var result serverutils.BaseResponse[dto.LoginResponse]
		json.NewDecoder(resp.Body).Decode(&result)

		assert.True(t, result.Success)
		assert.NotEmpty(t, result.Data.AccessToken)
		assert.Equal(t, "admin", result.Data.User.Role)
	})

	t.Run("Login as Regular User denied", func(t *testing.T) {
		reqBody := dto.LoginRequest{
			Email:    "testuser@example.com",
			Password: "admin123",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req, -1)

		// Expect 401 Unauthorized
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("Invalid Password", func(t *testing.T) {
		reqBody := dto.LoginRequest{
			Email:    "testadmin@example.com",
			Password: "wrongpassword",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req, -1)

		assert.Equal(t, 401, resp.StatusCode)
	})
}
