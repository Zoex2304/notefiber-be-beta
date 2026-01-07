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

	// Check for Plans named 'Pro Plan Cancel%'
	var planIds []string
	db.Raw("SELECT id FROM subscription_plans WHERE name LIKE 'Pro Plan Cancel%'").Scan(&planIds)

	log.Printf("Found %d plans matching 'Pro Plan Cancel%%'", len(planIds))

	if len(planIds) > 0 {
		// Count subs
		var subCount int64
		db.Table("user_subscriptions").Where("plan_id IN ?", planIds).Count(&subCount)
		log.Printf("Found %d subscriptions linked to these plans", subCount)

		if subCount > 0 {
			log.Println("Deleting orphaned subscriptions first...")
			// Delete subs explicitly
			db.Exec("DELETE FROM user_subscriptions WHERE plan_id IN ?", planIds)
		}

		log.Println("Deleting plans...")
		db.Exec("DELETE FROM subscription_plans WHERE id IN ?", planIds)
		log.Println("Done.")
	} else {
		log.Println("No plans found.")
	}
}
