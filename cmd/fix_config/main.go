package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	fmt.Println("=== Fix: Updating AI Configuration Threshold ===\n")

	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Printf("⚠️  Warning: Could not load .env file: %v\n", err)
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		fmt.Println("❌ DB_CONNECTION_STRING not set in environment")
		return
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Printf("❌ Database connection failed: %v\n", err)
		return
	}

	// Update the threshold
	targetValue := "0.35"
	key := "rag_similarity_threshold"

	fmt.Printf("🔄 Updating '%s' to '%s'...\n", key, targetValue)

	result := db.Table("ai_configurations").
		Where("key = ?", key).
		Update("value", targetValue)

	if result.Error != nil {
		fmt.Printf("❌ Update failed: %v\n", result.Error)
		return
	}

	if result.RowsAffected == 0 {
		fmt.Println("⚠️  No rows affected. Does the key exist?")
		// Try creating it if it doesn't exist? (Optional, but strict update usually preferred if we expect it to exist)
	} else {
		fmt.Println("✅ Successfully updated configuration!")
	}

	// Verify the change
	type AiConfig struct {
		Key   string
		Value string
	}
	var config AiConfig
	db.Table("ai_configurations").Where("key = ?", key).First(&config)
	fmt.Printf("   Current Value in DB: %s\n", config.Value)

	fmt.Println("\n=== Fix Complete ===")
}
