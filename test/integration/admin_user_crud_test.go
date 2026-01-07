package integration

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
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

func TestAdminUserCRUD(t *testing.T) {
	// Setup
	if err := godotenv.Load("../../.env"); err != nil {
		t.Logf("Warning: Could not load ../../.env: %v", err)
		// Fix for middleware mismatch if .env missing
		os.Setenv("JWT_SECRET", "default_secret")
	}
	cfg := config.Load()

	db, err := database.NewGormDBFromDSN(cfg.Database.Connection)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	container := bootstrap.NewContainer(db, cfg)
	srv := server.New(cfg, container)
	app := srv.GetApp()

	// 1. Seed Admin for Auth
	adminPass := "admin123"
	adminHash, _ := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	adminHashStr := string(adminHash)

	adminId := uuid.New()
	adminUser := &entity.User{
		Id:            adminId,
		Email:         "crudadmin@example.com",
		FullName:      "CRUD Admin",
		PasswordHash:  &adminHashStr,
		Role:          entity.UserRoleAdmin,
		Status:        entity.UserStatusActive, // Must be active
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Create Admin Helper
	// We need to bypass API for login or use login helper if we had one.
	// For simplicity, we create user in DB, then login via API to get token.

	db.Create(adminUser)
	defer db.Delete(&entity.User{}, adminId)

	// Login to get token
	loginReq := dto.LoginRequest{
		Email:    "crudadmin@example.com",
		Password: "admin123",
	}
	loginBody, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(string(loginBody)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)

	var loginRes serverutils.BaseResponse[dto.LoginResponse]
	json.NewDecoder(resp.Body).Decode(&loginRes)
	token := loginRes.Data.AccessToken
	assert.NotEmpty(t, token, "Admin token should not be empty")

	// 2. Test Case: Update User
	t.Run("Update User Details", func(t *testing.T) {
		// Create Target User
		targetId := uuid.New()
		targetUser := &entity.User{
			Id:            targetId,
			Email:         "target@example.com",
			FullName:      "Original Name",
			Role:          entity.UserRoleUser,
			Status:        entity.UserStatusActive,
			EmailVerified: true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		db.Create(targetUser)
		defer db.Delete(&entity.User{}, targetId) // Cleanup in case delete test fails

		// Update Request
		updateReq := dto.AdminUpdateUserRequest{
			FullName: "Updated Name",
			Role:     "admin", // Promote to admin
		}
		updateBody, _ := json.Marshal(updateReq)

		req := httptest.NewRequest("PUT", "/api/admin/users/"+targetId.String(), strings.NewReader(string(updateBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token) // Use Token

		resp, _ := app.Test(req, -1)
		assert.Equal(t, 200, resp.StatusCode)

		var updateRes serverutils.BaseResponse[dto.UserProfileResponse]
		json.NewDecoder(resp.Body).Decode(&updateRes)

		assert.Equal(t, "Updated Name", updateRes.Data.FullName)
		assert.Equal(t, "admin", updateRes.Data.Role)

		// Verify in DB
		var dbUser entity.User
		db.First(&dbUser, targetId)
		assert.Equal(t, "Updated Name", dbUser.FullName)
		assert.Equal(t, entity.UserRoleAdmin, dbUser.Role)
	})

	// 3. Test Case: Delete User
	t.Run("Delete User", func(t *testing.T) {
		// Create Victim User
		victimId := uuid.New()
		victimUser := &entity.User{
			Id:        victimId,
			Email:     "victim@example.com",
			FullName:  "Victim Name",
			Role:      entity.UserRoleUser,
			Status:    entity.UserStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.Create(victimUser)

		req := httptest.NewRequest("DELETE", "/api/admin/users/"+victimId.String(), nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, _ := app.Test(req, -1)

		if resp.StatusCode != 200 {
			var errRes serverutils.BaseResponse[any]
			json.NewDecoder(resp.Body).Decode(&errRes)
			fmt.Printf("Delete Status: %d, Msg: %s\n", resp.StatusCode, errRes.Message)
		}
		assert.Equal(t, 200, resp.StatusCode)

		// Verify in DB (Should be deleted - Hard or Soft)
		var result struct {
			Id        uuid.UUID
			DeletedAt *time.Time
		}
		// Using Raw to ensure we see the row state directly
		db.Raw("SELECT id, deleted_at FROM users WHERE id = ?", victimId).Scan(&result)

		if result.Id == uuid.Nil {
			// Row not found (Hard Delete) - Success
			return
		}
		// Row found (Soft Delete) - Check deleted_at
		assert.NotNil(t, result.DeletedAt, "User row exists but deleted_at is nil")
	})
}
