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

	log.Println("Starting Cleanup of Test Artifacts...")

	// 1. Find and Delete Test Subscriptions first
	// Access via User Email pattern "test_cancel_%"
	// Or Plan Name "Pro Plan Cancel%"

	// Delete Subscriptions for Test Users
	result := db.Exec(`
		DELETE FROM user_subscriptions 
		WHERE user_id IN (SELECT id FROM users WHERE email LIKE 'test_cancel_%')
	`)
	if result.Error != nil {
		log.Printf("Error deleting user_subscriptions: %v", result.Error)
	} else {
		log.Printf("Deleted %d test user_subscriptions", result.RowsAffected)
	}

	// 2. Delete Test Users
	result = db.Exec("DELETE FROM users WHERE email LIKE 'test_cancel_%'")
	if result.Error != nil {
		log.Printf("Error deleting users: %v", result.Error)
	} else {
		log.Printf("Deleted %d test users", result.RowsAffected)
	}

	// 3. Delete Test Plans
	result = db.Exec("DELETE FROM subscription_plans WHERE name LIKE 'Pro Plan Cancel%'")
	if result.Error != nil {
		log.Printf("Error deleting subscription_plans: %v", result.Error)
	} else {
		log.Printf("Deleted %d test subscription_plans", result.RowsAffected)
	}

	log.Println("Cleanup Complete.")
}
