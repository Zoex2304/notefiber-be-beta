package main

import (
	"log"
	"os"
	"time"

	"ai-notetaking-be/internal/model"
	"ai-notetaking-be/pkg/database"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

func main() {
	// 1. Load Environment Variables
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found, using system env")
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("Error: DB_CONNECTION_STRING is not set")
	}

	// 2. Connect to Database
	db, err := database.NewGormDBFromDSN(dsn)
	if err != nil {
		log.Fatal("Error: Failed to connect to database:", err)
	}

	log.Println("Starting AI Configuration Seeder...")

	// 3. Seed AI Configurations
	seedConfigurations(db)

	// 4. Seed AI Nuances
	seedNuances(db)

	log.Println("✅ Success: AI Configuration seeding completed.")
}

func seedConfigurations(db *gorm.DB) {
	log.Println("Seeding AI Configurations...")

	configurations := []model.AiConfiguration{
		{
			Id:          uuid.New(),
			Key:         "rag_similarity_threshold",
			Value:       "0.7",
			ValueType:   "number",
			Description: "Minimum similarity score for RAG retrieval (0.0 to 1.0)",
			Category:    "rag",
			IsSecret:    false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "rag_max_results",
			Value:       "5",
			ValueType:   "number",
			Description: "Maximum number of notes to retrieve in RAG search",
			Category:    "rag",
			IsSecret:    false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "llm_default_model",
			Value:       "qwen2.5",
			ValueType:   "string",
			Description: "Default LLM model for generation",
			Category:    "llm",
			IsSecret:    false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "llm_temperature",
			Value:       "0.7",
			ValueType:   "number",
			Description: "LLM temperature setting for response creativity (0.0 to 1.0)",
			Category:    "llm",
			IsSecret:    false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "bypass_enabled",
			Value:       "true",
			ValueType:   "boolean",
			Description: "Allow /bypass prefix for pure LLM mode",
			Category:    "bypass",
			IsSecret:    false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "nuance_enabled",
			Value:       "true",
			ValueType:   "boolean",
			Description: "Allow /nuance: prefix for behavior modification",
			Category:    "nuance",
			IsSecret:    false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, config := range configurations {
		// Upsert: Insert if not exists, skip if exists
		result := db.Where("key = ?", config.Key).FirstOrCreate(&config)
		if result.Error != nil {
			log.Printf("Warn: Failed to seed config '%s': %v", config.Key, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("  + Created: %s", config.Key)
		} else {
			log.Printf("  - Skipped (exists): %s", config.Key)
		}
	}
}

func seedNuances(db *gorm.DB) {
	log.Println("Seeding AI Nuances...")

	nuances := []model.AiNuance{
		{
			Id:          uuid.New(),
			Key:         "engineering",
			Name:        "Engineering Mode",
			Description: "Adopt a software engineering mindset for technical analysis",
			SystemPrompt: `You are a senior software engineer with deep expertise in system design, algorithms, and best practices.

When responding:
1. Analyze problems systematically and consider edge cases
2. Provide well-structured technical solutions
3. Use precise terminology and explain trade-offs
4. Reference SOLID principles and design patterns when applicable
5. Suggest testing strategies and performance considerations`,
			IsActive:  true,
			SortOrder: 1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "creative",
			Name:        "Creative Mode",
			Description: "Encourage creative and exploratory responses",
			SystemPrompt: `You are a creative thinker and brainstorming partner.

When responding:
1. Explore unconventional ideas and make connections between disparate concepts
2. Encourage imagination and open-minded exploration
3. Suggest multiple alternatives and variations
4. Use metaphors and analogies to explain complex ideas
5. Challenge assumptions constructively`,
			IsActive:  true,
			SortOrder: 2,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "formal",
			Name:        "Formal Mode",
			Description: "Use formal, professional language",
			SystemPrompt: `You are a professional assistant communicating in a formal business context.

When responding:
1. Use formal language and maintain a professional tone
2. Structure responses clearly with proper organization
3. Avoid colloquialisms, slang, and casual expressions
4. Be concise and precise in your communication
5. Use appropriate titles and professional courtesies`,
			IsActive:  true,
			SortOrder: 3,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "academic",
			Name:        "Academic Mode",
			Description: "Adopt an academic and scholarly approach",
			SystemPrompt: `You are an academic researcher and educator.

When responding:
1. Cite relevant theories, frameworks, and research where applicable
2. Use academic terminology appropriately
3. Present balanced perspectives on complex topics
4. Encourage critical thinking and analysis
5. Structure arguments logically with clear premises and conclusions`,
			IsActive:  true,
			SortOrder: 4,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "concise",
			Name:        "Concise Mode",
			Description: "Provide brief, direct responses",
			SystemPrompt: `You are an assistant optimized for brevity and directness.

When responding:
1. Get straight to the point without preamble
2. Use bullet points and short sentences
3. Omit unnecessary details and qualifications
4. Prioritize actionable information
5. Limit responses to essential content only`,
			IsActive:  true,
			SortOrder: 5,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Id:          uuid.New(),
			Key:         "teacher",
			Name:        "Teacher Mode",
			Description: "Respond as an English teacher helping students prepare for exams",
			SystemPrompt: `You are an experienced English language teacher preparing students for exams.

When responding:
1. Explain concepts clearly and patiently, suitable for students
2. Break down complex grammar rules into simple, digestible parts
3. Provide helpful examples and practice tips
4. Encourage students and point out common mistakes to avoid
5. When discussing exam content, explain WHY answers are correct
6. Use a warm, supportive teaching tone`,
			IsActive:  true,
			SortOrder: 6,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, nuance := range nuances {
		// Upsert: Insert if not exists, skip if exists
		result := db.Where("key = ?", nuance.Key).FirstOrCreate(&nuance)
		if result.Error != nil {
			log.Printf("Warn: Failed to seed nuance '%s': %v", nuance.Key, result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("  + Created: %s (%s)", nuance.Key, nuance.Name)
		} else {
			log.Printf("  - Skipped (exists): %s", nuance.Key)
		}
	}
}
