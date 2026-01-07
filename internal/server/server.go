package server

import (
	"log"

	"ai-notetaking-be/internal/bootstrap"
	"ai-notetaking-be/internal/config"
	"ai-notetaking-be/internal/pkg/serverutils"

	"github.com/gofiber/contrib/otelfiber"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type Server struct {
	app       *fiber.App
	cfg       *config.Config
	container *bootstrap.Container
}

func New(cfg *config.Config, container *bootstrap.Container) *Server {
	// Initialize Fiber App
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024, // 10MB
		// Add ErrorHandler here if preferred inside config, but middleware below works too
	})

	// Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.App.CorsAllowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		ExposeHeaders:    "Content-Length, Content-Type, Authorization",
	}))

	// OpenTelemetry tracing middleware (traces all HTTP requests)
	app.Use(otelfiber.Middleware())

	app.Use(serverutils.ErrorHandlerMiddleware())

	// Static
	app.Static("/uploads", "./uploads")

	// Routes
	registerRoutes(app, container)

	return &Server{
		app:       app,
		cfg:       cfg,
		container: container,
	}
}

func (s *Server) GetApp() *fiber.App {
	return s.app
}

func (s *Server) Run() error {
	log.Printf("✅ Server is running on http://localhost:%s", s.cfg.App.Port)
	return s.app.Listen(":" + s.cfg.App.Port)
}

func registerRoutes(app *fiber.App, c *bootstrap.Container) {
	api := app.Group("/api")

	c.AuthController.RegisterRoutes(api)
	c.UserController.RegisterRoutes(api)
	c.OAuthController.RegisterRoutes(api)

	c.NotebookController.RegisterRoutes(api)
	c.NoteController.RegisterRoutes(api)
	c.ChatbotController.RegisterRoutes(api)

	c.PaymentController.RegisterRoutes(api)
	c.AdminController.RegisterRoutes(api)
	c.LocationController.RegisterRoutes(api)
	c.PlanController.RegisterRoutes(api, serverutils.JwtMiddleware)

	c.NotificationHandler.RegisterRoutes(api)
}
