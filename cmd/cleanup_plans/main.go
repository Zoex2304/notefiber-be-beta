package main

import (
	"log"
	"os"

	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/pkg/database"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system env")
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("DB_CONNECTION_STRING is not set")
	}

	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Removing 'Integration Plan' from subscription_plans...")

	// Delete Hard Delete or Soft Delete?
	// SubscriptionPlan doesnt have DeletedAt in model (checked in previous steps, usually).
	// Let's check model again.
	// Step 532: SubscriptionPlan struct has NO DeletedAt. So it is hard delete.

	result := db.Where("name = ?", "Integration Plan").Delete(&model.SubscriptionPlan{})
	if result.Error != nil {
		log.Fatalf("Failed to delete: %v", result.Error)
	}

	log.Printf("Deleted %d rows.", result.RowsAffected)
}
