# Dokumentasi: Core Libraries & Foundation (Backend)

> **Fokus Domain:** BACKEND  
> **Konteks:** Library dependencies, internal packages, dan mappers yang menjadi fondasi project  
> **Scope:** Go modules, external libraries, internal pkg structure, dan design patterns

---

## Alur Arsitektural (Scope: BACKEND)

```
â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
â”‚                          APPLICATION                              â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚  cmd/server    â”‚  cmd/migrate   â”‚  cmd/seeder                    â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚                       INTERNAL LAYERS                             â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚ controller  â”‚  service    â”‚ repository  â”‚ mapper                 â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚     dto     â”‚   entity    â”‚   model     â”‚ specification          â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚                        INTERNAL PKG                               â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚ serverutils â”‚   mailer    â”‚   logger    â”‚ config                 â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚                       EXTERNAL PKG                                â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚  chatbot    â”‚  embedding  â”‚                                      â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚                    EXTERNAL LIBRARIES                             â”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚ Fiber â”‚ GORM â”‚ JWT â”‚ PGVector â”‚ Zap â”‚ Midtrans â”‚ OAuth2 â”‚ ...    â”‚
â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
```

---

## A. External Libraries (go.mod)

### 1. Web Framework

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `github.com/gofiber/fiber/v2` | v2.52.8 | HTTP framework berbasis fasthttp | âš¡ 10x faster than net/http |
| `github.com/gofiber/contrib/otelfiber` | v1.0.10 | OpenTelemetry middleware | ðŸ” Distributed tracing |

---

### 2. Database & ORM

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `gorm.io/gorm` | v1.31.1 | ORM untuk PostgreSQL | ðŸ—„ï¸ Auto-migration, hooks, soft delete |
| `gorm.io/driver/postgres` | v1.6.0 | PostgreSQL driver | PostgreSQL-specific features |
| `github.com/jackc/pgx/v5` | v5.7.5 | Native PostgreSQL driver | ðŸš€ High-performance, pgbouncer support |
| `github.com/pgvector/pgvector-go` | v0.3.0 | Vector similarity search | ðŸ§  AI semantic search via cosine distance |

---

### 3. Authentication & Security

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `github.com/golang-jwt/jwt/v5` | v5.3.0 | JWT generation & validation | ðŸ” HS256 signing |
| `golang.org/x/crypto` | v0.44.0 | bcrypt password hashing | ðŸ”’ Cost factor: 10 (default) |
| `golang.org/x/oauth2` | v0.33.0 | OAuth2 client | ðŸ”— Google Sign-In |

---

### 4. Logging & Observability

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `go.uber.org/zap` | v1.27.1 | Structured logging | âš¡ Zero-allocation JSON logging |
| `gopkg.in/natefinch/lumberjack.v2` | v2.2.1 | Log rotation | ðŸ“ Max 10MB, 5 backups, 30 days |
| `go.opentelemetry.io/otel` | v1.39.0 | Distributed tracing | ðŸ“Š OTLP HTTP export |

---

### 5. Payment Gateway

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `github.com/midtrans/midtrans-go` | v1.3.8 | Midtrans Snap payment | ðŸ’³ Snap redirect, webhook handler |

---

### 6. Email Service

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `gopkg.in/gomail.v2` | v2.0.0 | SMTP email sending | ðŸ“§ HTML templates, attachments |

---

### 7. Validation

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `github.com/go-playground/validator/v10` | v10.27.0 | Struct validation | âœ… Tags: required, email, min, max |

---

### 8. Utilities

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `github.com/google/uuid` | v1.6.0 | UUID generation | V4 random UUIDs |
| `github.com/joho/godotenv` | v1.5.1 | Environment loading | Load .env files |
| `github.com/stretchr/testify` | v1.11.1 | Testing | assert, require, mock |

---

### 9. Async Processing

| Library | Version | Purpose | Note |
|---------|---------|---------|------|
| `github.com/ThreeDotsLabs/watermill` | v1.4.7 | Pub-sub messaging | ðŸ”„ Async embedding generation |

---

## B. Internal Layers (internal/)

### 1. `internal/bootstrap`

| Component | File | Note |
|-----------|------|------|
| Container | `container.go` | ðŸ“¦ **Dependency Injection container** - Wiring semua services dan controllers |
| Database | `database.go` | ðŸ—„ï¸ GORM + PostgreSQL connection setup |

```go
// container.go - Wiring dependencies
type Container struct {
    AuthController   controller.IAuthController
    NoteController   controller.INoteController
    AdminController  controller.IAdminController
    // ...
}
```

---

### 2. `internal/server`

| Component | File | Note |
|-----------|------|------|
| Server | `server.go` | ðŸŒ **Fiber app setup** - Route registration, middleware, error handling |

```go
// Middleware chain
app.Use(cors.New())
app.Use(logger.New())
api := app.Group("/api")
```

---

### 3. `internal/controller`

| Controller | File | Note |
|------------|------|------|
| AuthController | `auth_controller.go` | ðŸ” Login, Register, OAuth, Password reset |
| NoteController | `note_controller.go` | ðŸ“ CRUD notes + semantic search |
| NotebookController | `notebook_controller.go` | ðŸ“ CRUD notebooks |
| UserController | `user_controller.go` | ðŸ‘¤ Profile, refund request |
| PaymentController | `payment_controller.go` | ðŸ’³ Checkout, webhook |
| AdminController | `admin_controller.go` | âš™ï¸ Admin dashboard + CRUD management |
| ChatbotController | `chatbot_controller.go` | ðŸ¤– AI chat dengan RAG |
| OAuthController | `oauth_controller.go` | ðŸ”— Google OAuth flow |
| PlanController | `plan_controller.go` | ðŸ“‹ Subscription plans |

> **Note:** Setiap controller implements interface dan register routes.

---

### 4. `internal/service`

| Service | File | Note |
|---------|------|------|
| AuthService | `auth_service.go` | ðŸ” Auth logic + JWT + bcrypt |
| NoteService | `note_service.go` | ðŸ“ Note CRUD + embedding trigger |
| NotebookService | `notebook_service.go` | ðŸ“ Notebook CRUD |
| UserService | `user_service.go` | ðŸ‘¤ Profile + refund request |
| PaymentService | `payment_service.go` | ðŸ’³ Checkout + webhook + subscription |
| AdminService | `admin_service.go` | âš™ï¸ Admin CRUD + approval |
| ChatbotService | `chatbot_service.go` | ðŸ¤– RAG + Ollama + usage tracking |
| OAuthService | `oauth_service.go` | ðŸ”— Google OAuth + user creation |
| PlanService | `plan_service.go` | ðŸ“‹ Limit enforcement |
| PublisherService | `publisher_service.go` | ðŸ”„ Async embedding via Watermill |

> **Note:** Services menggunakan Unit of Work untuk transaction management.

---

### 5. `internal/repository`

| Subdirectory | Note |
|--------------|------|
| `contract/` | ðŸ“œ **Interface definitions** - Repository contracts |
| `implementation/` | âš™ï¸ **GORM implementations** - Database queries |
| `unitofwork/` | ðŸ”„ **Transaction management** - Begin, Commit, Rollback |
| `specification/` | ðŸ” **Query builders** - Dynamic WHERE clauses |

---

### 6. `internal/entity`

| Entity | File | Note |
|--------|------|------|
| User | `user_entity.go` | ðŸ‘¤ Domain user dengan Role/Status enum |
| Note | `note_entity.go` | ðŸ“ Domain note |
| Notebook | `notebook_entity.go` | ðŸ“ Domain notebook |
| SubscriptionPlan | `subscription_entity.go` | ðŸ“‹ Plan dengan limits |
| UserSubscription | `subscription_entity.go` | ðŸŽ« User's active subscription |
| ChatSession | `chat_session_entity.go` | ðŸ’¬ AI chat session |
| ChatMessage | `chat_message_entity.go` | ðŸ’¬ Chat messages |
| NoteEmbedding | `note_embedding_entity.go` | ðŸ§  Vector embeddings |
| Refund | `refund_entity.go` | ðŸ’° Refund requests |

> **Note:** Entities adalah **domain objects** - bersih tanpa GORM tags.

---

### 7. `internal/model`

| Model | File | Note |
|-------|------|------|
| User | `user_model.go` | ðŸ—„ï¸ GORM model dengan tags |
| Note | `note_model.go` | ðŸ—„ï¸ GORM model |
| Notebook | `notebook_model.go` | ðŸ—„ï¸ GORM model |
| SubscriptionPlan | `subscription_model.go` | ðŸ—„ï¸ GORM model |
| NoteEmbedding | `note_embedding_model.go` | ðŸ§  pgvector.Vector type |
| ... | ... | ... |

> **Note:** Models adalah **persistence objects** - dengan GORM tags untuk DB mapping.

---

### 8. `internal/dto`

| DTO Group | File | Note |
|-----------|------|------|
| Auth | `auth_payment_dto.go` | ðŸ” Login, Register, JWT response |
| User | `user_dto.go` | ðŸ‘¤ Profile, subscription info |
| Note | `note_dto.go` | ðŸ“ CRUD request/response |
| Notebook | `notebook_dto.go` | ðŸ“ CRUD request/response |
| Admin | `admin_dto.go` | âš™ï¸ Admin operations |
| Chatbot | `chatbot_dto.go` | ðŸ¤– Chat request/response |
| Refund | `refund_dto.go` | ðŸ’° Refund request/response |

> **Note:** DTOs adalah **API contracts** - validasi dengan `validate` tags.

---

### 9. `internal/constant`

| Constant | File | Note |
|----------|------|------|
| Chatbot | `chatbot_constant.go` | ðŸ¤– System prompts, Ollama config |

> **Note:** Constants untuk magic values dan configuration.

---

## C. Mapper Layer (internal/mapper)

> **Note:** Mappers bertanggung jawab untuk **konversi bidirectional** antara Entity (domain) dan Model (persistence).

### Complete Mapper List

| Mapper | File | Converts | Note |
|--------|------|----------|------|
| **UserMapper** | `user_mapper.go` | User, PasswordResetToken, UserProvider, EmailVerificationToken, UserRefreshToken | ðŸ‘¤ Handles Role/Status enum conversion |
| **NoteMapper** | `note_mapper.go` | Note | ðŸ“ Handles gorm.DeletedAt â†” *time.Time |
| **NotebookMapper** | `notebook_mapper.go` | Notebook | ðŸ“ Handles soft delete |
| **ChatMapper** | `chat_mapper.go` | ChatSession, ChatMessage, ChatMessageRaw | ðŸ’¬ AI chat entities |
| **SubscriptionMapper** | `subscription_mapper.go` | SubscriptionPlan, UserSubscription | ðŸ“‹ Plan & subscription |
| **BillingMapper** | `billing_mapper.go` | Payment transactions | ðŸ’³ Payment records |
| **FeatureMapper** | `feature_mapper.go` | Feature catalog | âœ¨ Plan features |
| **NoteEmbeddingMapper** | `note_embedding_mapper.go` | NoteEmbedding | ðŸ§  pgvector.Vector â†” []float32 |

### Mapper Pattern

```go
// user_mapper.go
type UserMapper struct{}

func NewUserMapper() *UserMapper {
    return &UserMapper{}
}

// Entity â†’ Model (untuk write ke DB)
func (m *UserMapper) ToModel(u *entity.User) *model.User {
    return &model.User{
        Id:           u.Id,
        Email:        u.Email,
        Role:         string(u.Role),    // Enum â†’ string
        Status:       string(u.Status),  // Enum â†’ string
        // ...
    }
}

// Model â†’ Entity (untuk read dari DB)
func (m *UserMapper) ToEntity(u *model.User) *entity.User {
    return &entity.User{
        Id:     u.Id,
        Email:  u.Email,
        Role:   entity.UserRole(u.Role),    // string â†’ Enum
        Status: entity.UserStatus(u.Status), // string â†’ Enum
        // ...
    }
}

// Batch conversions
func (m *UserMapper) ToEntities(users []*model.User) []*entity.User
func (m *UserMapper) ToModels(users []*entity.User) []*model.User
```

### Special Mappings

| Type | Entity | Model | Note |
|------|--------|-------|------|
| Soft Delete | `*time.Time` | `gorm.DeletedAt` | Nullable timestamp |
| Enums | `entity.UserRole` | `string` | Type-safe enum |
| Vectors | `[]float32` | `pgvector.Vector` | PGVector type |

---

## D. Internal Packages (internal/pkg)

### 1. `pkg/serverutils`

| File | Note |
|------|------|
| `response.go` | ðŸ“¤ **Standard API response** - SuccessResponse, ErrorResponse |
| `jwt_middleware.go` | ðŸ” **JWT middleware** - Extract & validate token |
| `validator.go` | âœ… **Request validation** - Struct validation wrapper |

```go
// Usage
ctx.JSON(serverutils.SuccessResponse("Success", data))
ctx.JSON(serverutils.ErrorResponse(400, "Bad request"))
```

---

### 2. `pkg/mailer`

| File | Note |
|------|------|
| `email_service.go` | ðŸ“§ **SMTP abstraction** - OTP, password reset emails |

```go
type IEmailService interface {
    SendOTP(to, otp string) error
    SendPasswordReset(to, token string) error
}
```

---

### 3. `pkg/logger`

| File | Note |
|------|------|
| `zap_logger.go` | ðŸ“Š **Structured logging** - Zap + Lumberjack + Admin read |

```go
type ILogger interface {
    Debug(module, message string, details map[string]interface{})
    Info(module, message string, details map[string]interface{})
    Warn(module, message string, details map[string]interface{})
    Error(module, message string, details map[string]interface{})
    GetLogs(level string, limit, offset int) ([]LogEntry, error)  // Admin read
    GetLogById(id string) (*LogEntry, error)
}
```

---

### 4. `pkg/config`

| File | Note |
|------|------|
| `config.go` | âš™ï¸ **App configuration** - Environment variables loader |

---

## E. External Packages (pkg/)

### 1. `pkg/chatbot`

| File | Note |
|------|------|
| `ollama.go` | ðŸ¤– **Ollama LLM client** - Chat completion, RAG decision |

```go
func GetOllamaResponse(ctx context.Context, history []*ChatHistory) (string, error)
func DecideToUseRAGWithOllama(ctx context.Context, history []*ChatHistory) (bool, error)
```

---

### 2. `pkg/embedding`

| File | Note |
|------|------|
| `gemini.go` | ðŸ§  **Gemini Embedding API** - Vector generation |

```go
func GetGeminiEmbedding(apiKey, text, taskType string) (*EmbeddingResponse, error)
// TaskType: "RETRIEVAL_DOCUMENT" atau "RETRIEVAL_QUERY"
```

---

## F. Library Dependency Graph (PlantUML)

```plantuml
@startuml Library Dependency Graph

skinparam packageStyle rectangle
skinparam shadowing false
skinparam defaultFontName Arial

package "Entry Points" {
    [cmd/server] as S
    [cmd/migrate] as M
}

package "Application Layer" {
    [Controllers] as C
    [Services] as SV
}

package "Data Layer" {
    [Repositories] as R
    [Unit of Work] as UoW
    [Mappers] as MAP
}

package "Domain Layer" {
    [Entities] as E
    [DTOs] as DTO
    [Models] as MOD
}

package "Internal Pkg" {
    [serverutils] as SU
    [logger] as LOG
    [mailer] as MAIL
    [config] as CFG
}

package "External Pkg" {
    [chatbot] as CB
    [embedding] as EMB
}

package "External Libs" {
    [Fiber] as FB
    [GORM] as GORM
    [golang-jwt] as JWT
    [Zap Logger] as ZAP
    [Midtrans] as MID
    [PGVector] as PGV
    [Gomail] as GM
    [Validator] as VAL
    [OAuth2] as OA
}

S --> C
M --> GORM
C --> FB
C --> SV
C --> SU
C --> DTO
SV --> UoW
SV --> JWT
SV --> LOG
SV --> MAIL
SV --> CB
SV --> EMB
UoW --> R
R --> GORM
R --> MAP
R --> PGV
MAP --> E
MAP --> MOD
SV --> MID
SU --> VAL
MAIL --> GM
SV --> OA
S --> CFG

@enduml
```

---

## G. Design Patterns Summary

| Pattern | Location | Note |
|---------|----------|------|
| **Repository** | `repository/` | ðŸ“œ Abstraksi data access |
| **Unit of Work** | `unitofwork/` | ðŸ”„ Transaction management |
| **Specification** | `specification/` | ðŸ” Dynamic query building |
| **Mapper** | `mapper/` | ðŸ”€ Entity â†” Model conversion |
| **Dependency Injection** | `bootstrap/` | ðŸ“¦ Wiring via Container |
| **Layered Architecture** | `internal/` | ðŸ›ï¸ Controller â†’ Service â†’ Repository |

---

## H. Environment Variables

| Variable | Library | Note |
|----------|---------|------|
| `DATABASE_URL` | GORM | PostgreSQL connection |
| `JWT_SECRET` | golang-jwt | Token signing key |
| `GOOGLE_CLIENT_ID` | oauth2 | OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | oauth2 | OAuth client secret |
| `GOOGLE_GEMINI_API_KEY` | pkg/embedding | Gemini API key |
| `MIDTRANS_SERVER_KEY` | midtrans-go | Payment API key |
| `MIDTRANS_IS_PRODUCTION` | midtrans-go | Sandbox/Production |
| `SMTP_HOST` | gomail | Email server |
| `SMTP_PORT` | gomail | Email port |
| `SMTP_USERNAME` | gomail | Email auth |
| `SMTP_PASSWORD` | gomail | Email password |
| `OLLAMA_BASE_URL` | pkg/chatbot | LLM server URL |
| `FRONTEND_URL` | oauth2, midtrans | Callback URL |

---

## I. Version Requirements

| Requirement | Version | Note |
|-------------|---------|------|
| Go | 1.24.0+ | Latest Go version |
| PostgreSQL | 14+ | Required for pgvector |
| pgvector | 0.5.0+ | Vector extension |
| Ollama | Latest | Local LLM server |

---

*Dokumen ini di-generate dalam mode READ-ONLY tanpa modifikasi terhadap kode sumber.*
