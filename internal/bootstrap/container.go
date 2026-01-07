package bootstrap

import (
	"context"
	"log"

	"ai-notetaking-be/internal/config"
	"ai-notetaking-be/internal/controller"
	"ai-notetaking-be/internal/handler"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/pkg/mailer"
	"ai-notetaking-be/internal/repository/implementation"
	"ai-notetaking-be/internal/repository/memory"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/internal/service"
	"ai-notetaking-be/internal/websocket"
	"ai-notetaking-be/pkg/admin/aiconfig"
	"ai-notetaking-be/pkg/admin/dashboard"
	adminEvents "ai-notetaking-be/pkg/admin/events"
	"ai-notetaking-be/pkg/admin/feature"
	"ai-notetaking-be/pkg/admin/plan"
	"ai-notetaking-be/pkg/admin/refund"
	"ai-notetaking-be/pkg/admin/subscription"
	"ai-notetaking-be/pkg/admin/usage"
	"ai-notetaking-be/pkg/admin/user"
	"ai-notetaking-be/pkg/embedding"
	"ai-notetaking-be/pkg/embedding/jina"
	"ai-notetaking-be/pkg/llm/factory"

	pktNats "ai-notetaking-be/pkg/nats"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	// Controllers
	NotebookController controller.INotebookController
	NoteController     controller.INoteController
	UserController     controller.IUserController
	AuthController     controller.IAuthController
	OAuthController    controller.IOAuthController
	AdminController    controller.IAdminController
	PaymentController  controller.IPaymentController
	ChatbotController  controller.IChatbotController
	LocationController controller.ILocationController
	PlanController     controller.PlanController

	// Background Services (Exposed for main.go to run)
	ConsumerService service.IConsumerService

	// WebSockets & Notification
	NotificationHandler *handler.NotificationHandler
	WebSocketHub        *websocket.Hub
}

func NewContainer(db *gorm.DB, cfg *config.Config) *Container {
	// 1. Core Facades
	// 1. Core Facades
	uowFactory := unitofwork.NewRepositoryFactory(db)
	sysLogger := logger.NewZapLogger(cfg.App.LogFilePath, cfg.App.Environment == "production")

	emailService := mailer.NewEmailService(
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Email,
		cfg.SMTP.Password,
		cfg.SMTP.SenderName,
	)

	// 2. Event Bus
	watermillLogger := watermill.NewStdLogger(false, false)
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{},
		watermillLogger,
	)

	// 3. Services
	// Initialize Embedding Provider based on Config
	var embeddingProvider embedding.EmbeddingProvider
	if cfg.Ai.EmbeddingProvider == "ollama" {
		embeddingProvider = embedding.NewOllamaProvider(
			cfg.Ai.OllamaBaseURL,
			cfg.Ai.OllamaModel,
		)
		log.Printf("[INFO] Using Embedding Provider: OLLAMA (%s)", cfg.Ai.OllamaModel)
	} else if cfg.Ai.EmbeddingProvider == "jina" {
		embeddingProvider = jina.NewJinaProvider(cfg.Keys.Jina)
		log.Printf("[INFO] Using Embedding Provider: JINA AI")
	} else {
		embeddingProvider = embedding.NewGeminiProvider(cfg.Keys.GoogleGemini)
		log.Printf("[INFO] Using Embedding Provider: GEMINI")
	}

	// Initialize LLM Provider based on Config
	llmProvider, err := factory.NewLLMProvider(
		cfg.Ai.LLMProvider,
		cfg.Ai.LLMModel,
		cfg.Ai.OllamaBaseURL,
		cfg.Keys.HuggingFace,
	)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize LLM Provider: %v", err)
	}
	log.Printf("[INFO] Using LLM Provider: %s (%s)", cfg.Ai.LLMProvider, cfg.Ai.LLMModel)

	// Initialize In-Memory Session Storage
	sessionRepo := memory.NewSessionRepository()

	// 2.5 Infrastructure (Moved up for dependency injection)
	// NATS
	natsPub, err := pktNats.NewPublisher(cfg.App.NatsURL)
	if err != nil {
		log.Printf("[WARN] Failed to connect to NATS Publisher: %v", err)
	}
	natsSub, err := pktNats.NewSubscriber(cfg.App.NatsURL)
	if err != nil {
		log.Printf("[WARN] Failed to connect to NATS Subscriber: %v", err)
	}

	// Redis
	opt, err := redis.ParseURL(cfg.App.RedisURL)
	if err != nil {
		log.Printf("[WARN] Failed to parse Redis URL: %v. Using direct Addr", err)
		opt = &redis.Options{
			Addr: cfg.App.RedisURL,
		}
	}
	rdb := redis.NewClient(opt)
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Printf("[WARN] Failed to connect to Redis: %v", err)
	}

	// WebSocket Hub
	wsLogger := logger.NewIsolatedLogger("logs/notification.log")
	wsHub := websocket.NewHub(rdb, wsLogger)
	go wsHub.Run()

	publisherService := service.NewPublisherService(cfg.Keys.ExampleTopic, pubSub)
	consumerService := service.NewConsumerService(
		pubSub,
		cfg.Keys.ExampleTopic,
		uowFactory,
		embeddingProvider, // Injected
	)

	userService := service.NewUserService(uowFactory, natsPub)
	authService := service.NewAuthService(uowFactory, emailService, natsPub)
	oauthService := service.NewOAuthService(uowFactory)

	notebookService := service.NewNotebookService(uowFactory, publisherService)
	noteService := service.NewNoteService(
		uowFactory,
		publisherService,
		embeddingProvider, // Injected
		natsPub,
	)

	chatbotService := service.NewChatbotService(
		uowFactory,
		embeddingProvider, // Injected
		llmProvider,       // Injected
		sessionRepo,       // Injected
	)
	paymentService := service.NewPaymentService(uowFactory, natsPub)

	// Admin Domain Components
	adminEventPublisher := adminEvents.NewNatsPublisher(natsPub, sysLogger)
	userManager := user.NewManager(sysLogger, adminEventPublisher)
	subscriptionManager := subscription.NewManager(sysLogger)
	planManager := plan.NewManager()
	featureManager := feature.NewManager()
	refundProcessor := refund.NewProcessor(sysLogger, adminEventPublisher)
	usageTracker := usage.NewTracker(sysLogger, adminEventPublisher)
	dashboardAggregator := dashboard.NewAggregator(sysLogger)
	aiConfigManager := aiconfig.NewManager()

	adminService := service.NewAdminService(
		uowFactory,
		sysLogger,
		userManager,
		subscriptionManager,
		planManager,
		featureManager,
		refundProcessor,
		usageTracker,
		dashboardAggregator,
		adminEventPublisher,
		aiConfigManager,
	)

	locationService := service.NewLocationService(cfg.Keys.Geoapify, cfg.Keys.Binderbyte)
	planService := service.NewPlanService(uowFactory)

	// 3.5 Notification System Infrastructure
	// Notification Domain
	notifRepo := implementation.NewNotificationRepository(db)
	notifService := service.NewNotificationService(notifRepo, natsSub, wsHub, wsLogger) // Hub implements NotificationDelivery

	// Start Service (Worker)
	if natsSub != nil {
		go notifService.Start()
	}

	// Handler
	notifHandler := handler.NewNotificationHandler(notifService, natsPub, wsHub, wsLogger)

	// 4. Controllers
	// Note: We return the container with public fields for the server to register
	return &Container{
		NotificationHandler: notifHandler,
		WebSocketHub:        wsHub,
		NotebookController:  controller.NewNotebookController(notebookService),
		NoteController:      controller.NewNoteController(noteService),
		UserController:      controller.NewUserController(userService),
		AuthController:      controller.NewAuthController(authService),
		OAuthController:     controller.NewOAuthController(oauthService),
		AdminController:     controller.NewAdminController(adminService, authService),
		PaymentController:   controller.NewPaymentController(paymentService),
		ChatbotController:   controller.NewChatbotController(chatbotService),
		LocationController:  controller.NewLocationController(locationService),
		PlanController:      controller.NewPlanController(planService),

		ConsumerService: consumerService,
	}
}
