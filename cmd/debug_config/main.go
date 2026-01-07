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
	fmt.Println("=== Debug: AI Configurations Threshold Check ===\n")

	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Printf("⚠️  Warning: Could not load .env file: %v\n", err)
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		fmt.Println("❌ DB_CONNECTION_STRING not set in environment")
		return
	}

	fmt.Printf("📡 Connecting to database...\n")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Printf("❌ Database connection failed: %v\n", err)
		return
	}

	fmt.Println("✅ Connected!\n")

	// Query ai_configurations for threshold
	fmt.Println("📋 Checking ai_configurations table...")

	type AiConfig struct {
		Key   string
		Value string
	}

	var config AiConfig
	result := db.Table("ai_configurations").
		Select("key, value").
		Where("key = ?", "rag_similarity_threshold").
		First(&config)

	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			fmt.Println("⚠️  rag_similarity_threshold NOT FOUND in ai_configurations")
			fmt.Println("   → Will use default: 0.35")
		} else {
			fmt.Printf("❌ Query error: %v\n", result.Error)
		}
	} else {
		fmt.Printf("✅ Found: rag_similarity_threshold = %s\n", config.Value)
		fmt.Println()
		if config.Value == "0.7" {
			fmt.Println("⚠️  VALUE IS TOO HIGH!")
			fmt.Println("   0.7 is very strict and will filter most results.")
			fmt.Println("   Recommended: 0.35 - 0.5")
		}
	}

	// List all AI configurations
	fmt.Println("\n📋 All AI Configurations:")
	var allConfigs []AiConfig
	db.Table("ai_configurations").Select("key, value").Find(&allConfigs)

	for _, c := range allConfigs {
		fmt.Printf("   %s = %s\n", c.Key, c.Value)
	}

	// Check note_embeddings count
	fmt.Println("\n📋 Note Embeddings Status:")
	var count int64
	db.Table("note_embeddings").Count(&count)
	fmt.Printf("   Total embeddings: %d\n", count)

	// Check for specific user's notes
	userId := "a2b94f4c-b674-433b-90be-65a91a37e7a3"
	var noteCount int64
	db.Table("notes").Where("user_id = ?", userId).Count(&noteCount)
	fmt.Printf("   User's notes: %d\n", noteCount)

	var embeddingCount int64
	db.Table("note_embeddings").
		Joins("JOIN notes ON note_embeddings.note_id = notes.id").
		Where("notes.user_id = ?", userId).
		Count(&embeddingCount)
	fmt.Printf("   User's embeddings: %d\n", embeddingCount)

	if embeddingCount == 0 && noteCount > 0 {
		fmt.Println("\n⚠️  PROBLEM: User has notes but NO embeddings!")
		fmt.Println("   This means semantic search will always return empty.")
		fmt.Println("   Embeddings need to be generated for the user's notes.")
	}

	fmt.Println("\n=== Debug Complete ===")
}
