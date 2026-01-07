package main

import (
	"context"
	"log"

	"ai-notetaking-be/internal/bootstrap"
	"ai-notetaking-be/internal/config"
	"ai-notetaking-be/internal/server"
	"ai-notetaking-be/pkg/database"
)

func main() {
	// 0. Initialize Tracer - DISABLED
	// shutdownTracer := tracer.InitTracer()
	// defer shutdownTracer(context.Background())

	// 1. Load Configuration
	cfg := config.Load()

	// 2. Initialize Database
	gormDB, err := database.NewGormDBFromDSN(cfg.Database.Connection)
	if err != nil {
		log.Panicf("Unable to connect to GORM DB: %v", err)
	}

	// 3. Bootstrap Dependencies (Container)
	container := bootstrap.NewContainer(gormDB, cfg)

	// 4. Start Background Services
	// Note: In a larger app, we might use an errgroup or supervisor here
	go func() {
		log.Println("Background: Starting Consumer Service...")
		if err := container.ConsumerService.Consume(context.Background()); err != nil {
			log.Printf("Background Consumer Error: %v", err)
		}
	}()

	// 5. Initialize Server
	srv := server.New(cfg, container)

	// 6. Run Server
	log.Fatal(srv.Run())
}
