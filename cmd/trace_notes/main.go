package main

import (
	"log"
	"os"

	"ai-notetaking-be/pkg/database"

	"github.com/joho/godotenv"
)

type Note struct {
	ID      string `gorm:"type:uuid;primary_key"`
	Title   string
	Content string
	UserID  string `gorm:"type:uuid"`
}

func main() {
	// 1. Load Environment
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found, using system env")
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("Error: DB_CONNECTION_STRING is not set")
	}

	// 2. Connect to DB
	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal("Error: Failed to connect to database:", err)
	}

	userID := "a2b94f4c-b674-433b-90be-65a91a37e7a3"

	log.Printf("🔍 Checking notes for User ID: %s", userID)

	// 3. Query notes
	var englishNotes []Note
	if err := db.Where("user_id = ? AND LOWER(title) LIKE ?", userID, "%english%").Find(&englishNotes).Error; err != nil {
		log.Fatal("Query failed:", err)
	}
	log.Printf("Found %d notes matching 'english':", len(englishNotes))
	for i, n := range englishNotes {
		log.Printf("[%d] %s", i+1, n.Title)
	}

	var finalNotes []Note
	if err := db.Where("user_id = ? AND LOWER(title) LIKE ?", userID, "%final%").Find(&finalNotes).Error; err != nil {
		log.Fatal("Query failed:", err)
	}
	log.Printf("Found %d notes matching 'final':", len(finalNotes))
	for i, n := range finalNotes {
		log.Printf("[%d] %s", i+1, n.Title)
	}

	// 4. Query ALL notes for this user to see total count
	var total int64
	db.Model(&Note{}).Where("user_id = ?", userID).Count(&total)
	log.Printf("Total notes for user: %d", total)
}
