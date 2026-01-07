package main

import (
	"log"

	"ai-notetaking-be/internal/config"
	"ai-notetaking-be/pkg/database"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Printf("Warning: Could not load .env: %v", err)
	}
	cfg := config.Load()

	db, err := database.NewGormDBFromDSN(cfg.Database.Connection)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}

	// IDs identified as STUPIDITY (Test Artifacts)
	targetIds := []string{
		"106fba10-4387-4e28-aaa4-d65e4a6c6bd7", // Integration Plan
		"42138fd9-eaa6-4deb-89d9-59dfd232f135", // Pro Plan (Price 10)
		"2cba6159-985e-452c-95ec-4364f9e430b7", // Pro Plan (Price 10)
	}

	log.Printf("Targeting %d plans for deletion...", len(targetIds))

	tx := db.Begin()

	// 0. Get Subscription IDs related to these plans
	var subIds []string
	tx.Raw("SELECT id FROM user_subscriptions WHERE plan_id IN ?", targetIds).Scan(&subIds)
	log.Printf("Found %d subscriptions to delete.", len(subIds))

	if len(subIds) > 0 {
		// 1. Delete Refunds linked to subscriptions
		res := tx.Exec("DELETE FROM refunds WHERE subscription_id IN ?", subIds)
		if res.Error != nil {
			tx.Rollback()
			log.Fatalf("Error deleting refunds: %v", res.Error)
		}
		log.Printf("Deleted %d refunds.", res.RowsAffected)

		// 2. Delete Cancellations linked to subscriptions
		res = tx.Exec("DELETE FROM cancellations WHERE subscription_id IN ?", subIds)
		if res.Error != nil {
			tx.Rollback()
			log.Fatalf("Error deleting cancellations: %v", res.Error)
		}
		log.Printf("Deleted %d cancellations.", res.RowsAffected)

		// 3. Delete Subscriptions linked to these plans
		res = tx.Exec("DELETE FROM user_subscriptions WHERE id IN ?", subIds)
		if res.Error != nil {
			tx.Rollback()
			log.Fatalf("Error deleting subscriptions: %v", res.Error)
		}
		log.Printf("Deleted %d subscriptions.", res.RowsAffected)
	}

	// 4. Delete Plans
	result := tx.Exec("DELETE FROM subscription_plans WHERE id IN ?", targetIds)
	if result.Error != nil {
		tx.Rollback()
		log.Fatalf("Error deleting plans: %v", result.Error)
	}
	log.Printf("Deleted %d test plans.", result.RowsAffected)

	tx.Commit()
	log.Println("Cleanup Success.")
}
