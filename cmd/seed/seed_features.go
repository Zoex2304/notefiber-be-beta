package main

import (
	"log"
	"os"

	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/pkg/database"

	"github.com/joho/godotenv"
)

func main() {
	// Load Environment Variables
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

	log.Println("Seeding Feature Catalog...")

	// Define initial features for the master catalog
	features := []model.Feature{
		{Key: "ai_chat", Name: "AI Chat Assistant", Description: "Chat with AI about your notes using context-aware responses", Category: "ai", IsActive: true, SortOrder: 1},
		{Key: "semantic_search", Name: "Semantic Search", Description: "Search notes using natural language and find related content", Category: "ai", IsActive: true, SortOrder: 2},
		{Key: "unlimited_notebooks", Name: "Unlimited Notebooks", Description: "Create as many notebooks as you need", Category: "storage", IsActive: true, SortOrder: 3},
		{Key: "unlimited_notes", Name: "Unlimited Notes", Description: "No limits on the number of notes per notebook", Category: "storage", IsActive: true, SortOrder: 4},
		{Key: "priority_support", Name: "Priority Support", Description: "Get faster response times from our support team", Category: "support", IsActive: true, SortOrder: 5},
		{Key: "export_pdf", Name: "Export to PDF", Description: "Export your notes in PDF format", Category: "export", IsActive: true, SortOrder: 6},
		{Key: "offline_mode", Name: "Offline Mode", Description: "Access your notes even without internet connection", Category: "sync", IsActive: true, SortOrder: 7},
	}

	for _, f := range features {
		// Check if feature with this key already exists
		var existing model.Feature
		if err := db.Where("key = ?", f.Key).First(&existing).Error; err == nil {
			log.Printf("Feature '%s' already exists, skipping...", f.Key)
			continue
		}

		if err := db.Create(&f).Error; err != nil {
			log.Printf("Error creating feature '%s': %v", f.Key, err)
		} else {
			log.Printf("Created feature: %s (%s)", f.Name, f.Key)
		}
	}

	log.Println("Feature seeding completed!")

	log.Println("Seeding Notification Types...")
	SeedNotificationTypes(db)
}
