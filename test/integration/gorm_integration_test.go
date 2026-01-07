package integration

import (
	"context"
	"log"
	"os"
	"testing"

	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/database"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestGormConnection(t *testing.T) {
	// Load .env from root
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("No .env file found, using system env")
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		t.Skip("Skipping integration test: DB_CONNECTION_STRING not set")
	}

	gormDB, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	// Verify Wiring
	uowFactory := unitofwork.NewRepositoryFactory(gormDB)
	uow := uowFactory.NewUnitOfWork(context.Background())

	assert.NotNil(t, uow.UserRepository())
	assert.NotNil(t, uow.SubscriptionRepository())

	// Basic Ping
	sqlDB, _ := gormDB.DB()
	err = sqlDB.Ping()
	assert.NoError(t, err)
	t.Log("Successfully connected to DB and initialized UnitOfWork Factory")

	// Verify Data Access (implies columns exist)
	t.Run("Check User Repository", func(t *testing.T) {
		count, err := uow.UserRepository().Count(context.Background())
		assert.NoError(t, err)
		t.Logf("User count: %d", count)
	})

	t.Run("Check Note Embedding Repository", func(t *testing.T) {
		// Just check successful access, table should exist
		// Count implies table check
		count, err := uow.NoteEmbeddingRepository().Count(context.Background())
		assert.NoError(t, err)
		t.Logf("NoteEmbedding count: %d", count)
	})

	t.Run("Check Transactional Billing Subscription", func(t *testing.T) {
		// Mock Data
		userId := uuid.New() // Need valid user if FK exists on billing?
		// Note: BillingAddress might require valid UserID if FK exists.
		// Let's create a User first to be safe.
		user := &entity.User{
			Id:       userId,
			Email:    "test-integration-" + uuid.New().String() + "@example.com",
			FullName: "Integration Test User",
			Role:     "user",
			Status:   "active",
		}

		// Create Plan
		planId := uuid.New()
		plan := &entity.SubscriptionPlan{
			Id:            planId,
			Name:          "Integration Plan",
			Slug:          "integration-plan-" + uuid.New().String(),
			Price:         10.0,
			BillingPeriod: "monthly",
		}

		// Setup DB Data
		err := uow.UserRepository().Create(context.Background(), user)
		assert.NoError(t, err)
		err = uow.SubscriptionRepository().CreatePlan(context.Background(), plan)
		assert.NoError(t, err)

		// Transaction Test
		ctx := context.Background()
		err = uow.Begin(ctx)
		assert.NoError(t, err)
		defer uow.Rollback()

		billingId := uuid.New()
		billing := &entity.BillingAddress{
			Id:           billingId,
			UserId:       userId,
			FirstName:    "Test",
			LastName:     "User",
			Email:        "test@example.com",
			AddressLine1: "123 Street",
			City:         "Test City",
			State:        "Test State",
			PostalCode:   "12345",
			Country:      "Test Country",
		}

		err = uow.BillingRepository().Create(ctx, billing)
		assert.NoError(t, err)

		subId := uuid.New()
		sub := &entity.UserSubscription{
			Id:               subId,
			UserId:           userId,
			PlanId:           planId,
			BillingAddressId: &billingId, // Reference the ID just created
			Status:           "active",
			PaymentStatus:    "paid",
		}

		err = uow.SubscriptionRepository().CreateSubscription(ctx, sub)
		assert.NoError(t, err)

		err = uow.Commit()
		assert.NoError(t, err)

		t.Log("Successfully created Subscription with Billing in Transaction")
	})
}
