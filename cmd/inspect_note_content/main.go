package main

import (
	"fmt"
	"log"
	"os"

	"ai-notetaking-be/pkg/database"
	"ai-notetaking-be/pkg/lexical"

	"github.com/joho/godotenv"
)

type Note struct {
	ID      string `gorm:"type:uuid;primary_key"`
	Title   string
	Content string
	UserID  string `gorm:"type:uuid"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found, using system env")
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("Error: DB_CONNECTION_STRING is not set")
	}

	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal("Error: Failed to connect to database:", err)
	}

	// Fetch Note 2 "English Final Examination"
	noteID := "5336a88a-35bd-4a19-b7e0-363828359b79"
	var note Note
	if err := db.Where("id = ?", noteID).First(&note).Error; err != nil {
		log.Fatal("Note not found:", err)
	}

	fmt.Printf("🔍 INSPECTING NOTE: %s (%s)\n", note.Title, note.ID)
	fmt.Printf("Raw Content Length: %d bytes\n", len(note.Content))

	// Parse Content
	markdown := lexical.ParseContent(note.Content)

	fmt.Println("\n" + "─" + " PARSED MARKDOWN " + "─")
	fmt.Println(markdown)
	fmt.Println("─" + " END MARKDOWN " + "─")
}
