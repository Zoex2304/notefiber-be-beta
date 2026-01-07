package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"ai-notetaking-be/pkg/database"

	"github.com/joho/godotenv"
	"gorm.io/gorm/logger"
)

type SimpleNote struct {
	ID    string
	Title string
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env found")
	}
	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("DSN not set")
	}

	// SILENT DB
	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.Logger = db.Logger.LogMode(logger.Silent)

	// 1. List Users
	var users []map[string]interface{}
	if err := db.Table("users").Select("id, email").Scan(&users).Error; err != nil {
		log.Fatal("Failed reading users:", err)
	}

	fmt.Println("🔎 DATA DIVERGENCE AUDIT (Filtered)")
	fmt.Printf("Total Users found: %d\n", len(users))
	fmt.Println(strings.Repeat("=", 60))

	for _, u := range users {
		uid := u["id"].(string)
		email := u["email"]

		// 2. Check Match Counts
		var englishNotes []SimpleNote
		db.Table("notes").Select("id, title").
			Where("user_id = ? AND LOWER(title) LIKE ?", uid, "%english%").
			Scan(&englishNotes)

		var classNotes []SimpleNote
		db.Table("notes").Select("id, title").
			Where("user_id = ? AND LOWER(title) LIKE ?", uid, "%class fund%").
			Scan(&classNotes)

		var finalNotes []SimpleNote
		db.Table("notes").Select("id, title").
			Where("user_id = ? AND LOWER(title) LIKE ?", uid, "%final exam%").
			Scan(&finalNotes)

		// IF ALL EMPTY, SKIP
		if len(englishNotes) == 0 && len(classNotes) == 0 && len(finalNotes) == 0 {
			continue
		}

		fmt.Printf("👤 USER: %s (%v)\n", uid, email)
		fmt.Printf("   📝 'English' Matches:     %d  [Example: %v]\n", len(englishNotes), getTitle(englishNotes))
		fmt.Printf("   📝 'Class Fund' Matches:  %d  [Example: %v]\n", len(classNotes), getTitle(classNotes))
		fmt.Printf("   📝 'Final Exam' Matches:  %d  [Example: %v]\n", len(finalNotes), getTitle(finalNotes))

		fmt.Println(strings.Repeat("-", 60))
	}
}

func getTitle(notes []SimpleNote) string {
	if len(notes) > 0 {
		return "'" + notes[0].Title + "'"
	}
	return "None"
}
