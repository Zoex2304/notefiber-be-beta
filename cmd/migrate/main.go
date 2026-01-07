package main

import (
	"log"
	"os"

	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/pkg/database"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found, using system env")
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("Error: DB_CONNECTION_STRING is not set")
	}

	// 2. Connect to Database using existing GORM helpers
	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal("Error: Failed to connect to database:", err)
	}

	log.Println("Starting Authoritative GORM Migration...")

	// 3. Pre-Migration: Extensions & Enums (Things GORM AutoMigrate doesn't do perfectly)
	log.Println("Step 1: Setting up Extensions and Enums...")

	setupSQL := []string{
		// Extensions
		`CREATE EXTENSION IF NOT EXISTS pgcrypto;`,
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`,
		`CREATE EXTENSION IF NOT EXISTS vector;`,

		// Enums (Idempotent creation)
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ai_credit_transaction_type') THEN CREATE TYPE ai_credit_transaction_type AS ENUM ('grant', 'spend', 'refund', 'adjustment'); END IF; END $$;`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_period') THEN CREATE TYPE billing_period AS ENUM ('monthly', 'yearly'); END IF; END $$;`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'payment_status') THEN CREATE TYPE payment_status AS ENUM ('pending', 'success', 'failed', 'refunded'); END IF; END $$;`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'subscription_status') THEN CREATE TYPE subscription_status AS ENUM ('active', 'inactive', 'canceled', 'trial'); END IF; END $$;`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN CREATE TYPE user_role AS ENUM ('user', 'admin'); END IF; END $$;`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN CREATE TYPE user_status AS ENUM ('pending', 'active', 'suspended', 'deleted'); END IF; END $$;`,
	}

	for _, sql := range setupSQL {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("Warn: Failed to execute setup SQL: %v. Continuing...", err)
		}
	}

	// 4. AutoMigrate All Models (The Core Task)
	log.Println("Step 2: Running AutoMigrate for 23 Tables...")

	models := []interface{}{
		&model.User{},
		&model.PasswordResetToken{},
		&model.UserProvider{},
		&model.EmailVerificationToken{},
		&model.UserRefreshToken{},
		&model.Notebook{},
		&model.Note{},
		&model.NoteEmbedding{},
		&model.ChatSession{},
		&model.ChatMessage{},
		&model.ChatMessage{},
		&model.ChatMessageRaw{},
		&model.ChatMessageReference{}, // NEW: Persistent User References
		&model.ChatCitation{},         // NEW: RAG Citations
		&model.Feature{},
		&model.SubscriptionPlan{},
		&model.SubscriptionPlanFeature{}, // Explicit Join Table
		&model.UserSubscription{},
		&model.BillingAddress{},
		&model.UserBillingAddress{}, // Separate table in schema.sql
		&model.Refund{},
		&model.Cancellation{}, // NEW: Subscription cancellation requests
		&model.AiCreditTransaction{},
		&model.SystemLog{},
		&model.NotificationType{},
		&model.Notification{},
		&model.UserNotificationPreference{},
		// AI Configuration Tables (Phase 2)
		&model.AiConfiguration{},
		&model.AiNuance{},
	}

	// Migrate strictly
	if err := db.AutoMigrate(models...); err != nil {
		log.Fatalf("Error: AutoMigrate failed: %v", err)
	}

	// 5. Post-Migration: Views & Functions
	log.Println("Step 3: Creating Views and Functions...")

	postMigrationSQL := []string{
		// Function: set_current_timestamp_updated_at
		`CREATE OR REPLACE FUNCTION set_current_timestamp_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
		DECLARE _new_value TIMESTAMP WITH TIME ZONE;
		BEGIN
		  _new_value := now();
		  IF NEW.updated_at IS DISTINCT FROM _new_value THEN NEW.updated_at = _new_value; END IF;
		  RETURN NEW;
		END; $$;`,

		// View: semantic_searchable_notes
		`CREATE OR REPLACE VIEW semantic_searchable_notes AS
		 SELECT n.id AS note_id, n.title, n.content, ne.embedding_value AS embedding, n.user_id
		 FROM notes n JOIN note_embeddings ne ON n.id = ne.note_id
		 WHERE n.deleted_at IS NULL;`,

		// View: user_payment_history
		`CREATE OR REPLACE VIEW user_payment_history AS
		 SELECT us.user_id, u.full_name, sp.name AS plan_name, sp.price, us.payment_status, us.midtrans_transaction_id, us.created_at AS payment_date
		 FROM user_subscriptions us
		 JOIN users u ON us.user_id = u.id
		 JOIN subscription_plans sp ON us.plan_id = sp.id
		 ORDER BY us.created_at DESC;`,
	}

	for _, sql := range postMigrationSQL {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("Warn: Failed to execute post-migration SQL: %v", err)
		}
	}

	log.Println("✅ Success: Database migration completed successfully via GORM.")
}
