package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	SMTP     SMTPConfig
	Keys     APIKeys
	Ai       AIConfig
}

type AppConfig struct {
	Port               string
	BaseURL            string
	ClientURL          string
	Environment        string
	LogFilePath        string
	CorsAllowedOrigins string
	NatsURL            string
	RedisURL           string
}

type DatabaseConfig struct {
	Connection string
}

type SMTPConfig struct {
	Host       string
	Port       int
	Email      string
	Password   string
	SenderName string
}

type APIKeys struct {
	Geoapify     string
	Binderbyte   string
	GoogleGemini string
	ExampleTopic string // Embedding topic
	Ai           AIConfig
}

type AIConfig struct {
	EmbeddingProvider string // "gemini" or "ollama"
	OllamaBaseURL     string
	OllamaModel       string
	LLMProvider       string // "ollama", "openai", etc
	LLMModel          string // e.g. "llama3", "qwen2.5"
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found, usage system environment")
	}

	return &Config{
		App: AppConfig{
			Port:               getEnv("APP_PORT", "3000"),
			BaseURL:            getEnv("APP_BASE_URL", "http://localhost:3000"),
			ClientURL:          getEnv("CLIENT_URL", "http://localhost:5173"),
			Environment:        getEnv("GO_ENV", "development"),
			LogFilePath:        getEnv("LOG_FILE_PATH", "app.log.csv"),
			CorsAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
			NatsURL:            getEnv("NATS_URL", "nats://localhost:4222"),
			RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		},
		Database: DatabaseConfig{
			Connection: getEnv("DB_CONNECTION_STRING", ""),
		},
		SMTP: SMTPConfig{
			Host:       getEnv("SMTP_HOST", ""),
			Port:       getEnvAsInt("SMTP_PORT", 587),
			Email:      getEnv("SMTP_EMAIL", ""),
			Password:   getEnv("SMTP_PASSWORD", ""),
			SenderName: getEnv("SMTP_SENDER_NAME", "NoteFiber"),
		},
		Keys: APIKeys{
			Geoapify:     getEnv("GEOAPIFY_API_KEY", ""),
			Binderbyte:   getEnv("BINDERBYTE_API_KEY", ""),
			GoogleGemini: getEnv("GOOGLE_GEMINI_API_KEY", ""),
			ExampleTopic: getEnv("EMBED_NOTE_CONTENT_TOPIC_NAME", "EMBED_NOTE_CONTENT"),
		},
		Ai: AIConfig{
			EmbeddingProvider: getEnv("EMBEDDING_PROVIDER", "gemini"),
			OllamaBaseURL:     getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			OllamaModel:       getEnv("OLLAMA_EMBEDDING_MODEL", "nomic-embed-text"),
			LLMProvider:       getEnv("LLM_PROVIDER", "ollama"),
			LLMModel:          getEnv("LLM_MODEL", "llama3"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	strValue := getEnv(key, "")
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return fallback
}
