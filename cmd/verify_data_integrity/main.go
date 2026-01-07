package main

import (
	"log"
	"os"
	"strings"

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

	log.Printf("🔍 DATA INTEGRITY CHECK for User ID: %s", userID)

	// 3. Query all notes matching "English" or "Exam"
	var notes []Note
	if err := db.Where("user_id = ? AND (LOWER(title) LIKE ? OR LOWER(title) LIKE ?)", userID, "%english%", "%final exam%").Find(&notes).Error; err != nil {
		log.Fatal("Query failed:", err)
	}

	log.Printf("Found %d notes matching 'English' or 'Final Exam':", len(notes))
	for i, n := range notes {
		log.Println(strings.Repeat("─", 50))
		log.Printf("[%d] ID: %s", i+1, n.ID)
		log.Printf("    Title: '%s'", n.Title)
		log.Printf("    Length: %d bytes", len(n.Content))

		// Check for Question markers
		q1 := strings.Contains(n.Content, "Question 1")
		q6 := strings.Contains(n.Content, "Question 6")
		q8 := strings.Contains(n.Content, "Question 8")

		log.Printf("    Contains 'Question 1': %v", q1)
		log.Printf("    Contains 'Question 6': %v", q6)
		log.Printf("    Contains 'Question 8': %v", q8)

		// Check start/end of content
		if len(n.Content) > 100 {
			log.Printf("    Start: %.50s...", n.Content)
			log.Printf("    End:   ...%.50s", n.Content[len(n.Content)-50:])
		} else {
			log.Printf("    Content: %s", n.Content)
		}
	}
}
