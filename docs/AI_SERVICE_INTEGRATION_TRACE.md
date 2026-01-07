# 🔄 AI Service Integration Layer - Upstream to Downstream Trace

## Dokumentasi Path Alur Data (Trace Deep)

**Tanggal:** 28 December 2025  
**Aplikasi:** Note Fiber Backend - AI Service Integration Layer

---

## 📊 UPSTREAM → DOWNSTREAM ARCHITECTURE

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          HTTP REQUEST (Entry Point)                      │
│                    POST /chatbot/v1/send-chat                           │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 1️⃣  CONTROLLER LAYER (Handler Entry Point)                             │
│     📄 internal/controller/chatbot_controller.go                        │
│     ├─ IChatbotController Interface                                     │
│     ├─ SendChat(ctx *fiber.Ctx) error                                   │
│     │  ├─ Extract user_id from JWT (Locals)                             │
│     │  ├─ Parse request body (SendChatRequest)                          │
│     │  └─ Delegate to Service Layer                                     │
│     └─ Dependency: IChatbotService                                      │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 2️⃣  SERVICE LAYER (Business Logic & Orchestration)                      │
│     📄 internal/service/chatbot_service.go                              │
│                                                                          │
│     SendChat() Flow:                                                     │
│     ├─ 🔐 AUTHENTICATION & AUTHORIZATION                                │
│     │  ├─ Verify user_id ownership                                      │
│     │  └─ Check subscription status (Pro Plan Required)                 │
│     │                                                                    │
│     ├─ 📊 TOKEN USAGE ENFORCEMENT                                       │
│     │  ├─ Fetch user record from DB                                     │
│     │  ├─ Check daily usage reset (24-hour window)                      │
│     │  └─ Validate against AiChatDailyLimit                             │
│     │                                                                    │
│     ├─ 💾 CHAT SESSION VALIDATION                                       │
│     │  └─ Verify session exists AND belongs to user                     │
│     │                                                                    │
│     ├─ 🤖 AI DECISION LAYER (Use RAG or Direct Response)                │
│     │  ├─ Generate embedding untuk user query                           │
│     │  │  └─ embedding.GetGeminiEmbedding()                             │
│     │  │                                                                 │
│     │  ├─ Decide: Use RAG (Retrieval-Augmented Generation)?             │
│     │  │  └─ chatbot.DecideToUseRAGWithOllama()                         │
│     │  │                                                                 │
│     │  └─ IF useRAG == true:                                            │
│     │     ├─ Search similar notes dari DB                               │
│     │     │  └─ NoteEmbeddingRepository.SearchSimilar()                 │
│     │     └─ Attach references ke prompt                                │
│     │                                                                    │
│     ├─ 🧠 AI RESPONSE GENERATION                                        │
│     │  ├─ Build complete prompt dengan context                          │
│     │  ├─ Include chat history                                          │
│     │  └─ Send to LLM backend                                           │
│     │     └─ chatbot.GetOllamaResponse()                                │
│     │                                                                    │
│     └─ 💾 PERSISTENCE LAYER                                             │
│        ├─ Save user message (ChatMessage)                               │
│        ├─ Save raw message version (ChatMessageRaw)                     │
│        ├─ Save model response (ChatMessage)                             │
│        ├─ Update session title (jika first response)                    │
│        └─ Update user.AiDailyUsage counter                              │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 3️⃣  REPOSITORY LAYER (Data Access & Domain)                             │
│     📄 internal/repository/                                             │
│                                                                          │
│     A. Chat Domain Repositories:                                        │
│        ├─ ChatSessionRepository                                         │
│        │  ├─ Create() - Create new session                              │
│        │  ├─ FindOne() - Query by ID + ownership                        │
│        │  └─ Update() - Update session title                            │
│        │                                                                 │
│        ├─ ChatMessageRepository                                         │
│        │  ├─ Create() - Save user/model messages                        │
│        │  └─ FindAll() - Get history dengan ordering                    │
│        │                                                                 │
│        ├─ ChatMessageRawRepository                                      │
│        │  ├─ Create() - Save raw prompt/response                        │
│        │  └─ FindAll() - Get raw conversation history                   │
│        │                                                                 │
│     B. User Domain Repositories:                                        │
│        ├─ UserRepository                                                │
│        │  ├─ FindOne() - Get user by ID                                 │
│        │  └─ Update() - Update AiDailyUsage counter                     │
│        │                                                                 │
│     C. Subscription Repositories:                                       │
│        ├─ SubscriptionRepository                                        │
│        │  ├─ FindAllSubscriptions() - Get active plans                  │
│        │  └─ FindOnePlan() - Get plan details (AiChat settings)         │
│        │                                                                 │
│     D. Embedding/Vector Repositories:                                   │
│        └─ NoteEmbeddingRepository                                       │
│           └─ SearchSimilar(vector, topK, userId) - Semantic search      │
│                                                                          │
│     Unit of Work Pattern:                                               │
│        ├─ Transactional consistency untuk multiple entities             │
│        └─ Rollback on error                                             │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 4️⃣  AI INTEGRATION LAYER (Chatbot Providers)                            │
│     📄 pkg/chatbot/                                                     │
│                                                                          │
│     A. Ollama Integration (Local LLM)                                    │
│        ├─ ollama_chatbot.go                                             │
│        ├─ GetOllamaResponse(ctx, chatHistories)                         │
│        │  └─ HTTP POST ke Ollama server (localhost:11434)               │
│        ├─ DecideToUseRAGWithOllama(ctx, histories)                      │
│        │  └─ Query Ollama untuk determine RAG necessity                 │
│        └─ Response: LLM-generated text                                  │
│                                                                          │
│     B. Gemini Integration (Google AI)                                    │
│        ├─ gemini_chatbot.go                                             │
│        ├─ GetGeminiResponse(ctx, apiKey, histories)                     │
│        │  ├─ Transform messages ke Gemini format                        │
│        │  ├─ HTTP POST ke Google Gemini API                             │
│        │  └─ Parse response candidates                                  │
│        └─ Structured request/response formats                           │
│                                                                          │
│     C. Embedding Service (Vector Generation)                            │
│        └─ pkg/embedding/                                                │
│           └─ GetGeminiEmbedding(apiKey, text, task_type)                │
│              ├─ Generate vector representation                          │
│              └─ Used untuk semantic search & RAG                        │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 5️⃣  EXTERNAL AI SERVICES (Downstream Dependencies)                      │
│                                                                          │
│     🖥️  Ollama Server (Local LLM)                                        │
│     ├─ Endpoint: http://localhost:11434/api/generate                    │
│     ├─ Models: Llama2, Mistral, Neural-Chat, etc.                       │
│     ├─ Purpose: Generate responses, decide RAG usage                    │
│     └─ Latency: ~1-5 seconds per request                                │
│                                                                          │
│     ☁️  Google Gemini API                                                │
│     ├─ Endpoint: https://generativelanguage.googleapis.com/v1beta/...   │
│     ├─ Models: Gemini 1.5, Embedding API                                │
│     ├─ Purpose: Fallback response, generate embeddings                  │
│     └─ Auth: Google API Key                                             │
│                                                                          │
│     🗄️  PostgreSQL Database                                              │
│     ├─ Stores: chat_sessions, chat_messages, chat_message_raw           │
│     ├─ Stores: note_embeddings (vector column)                          │
│     ├─ Vector Search: pgvector extension                                │
│     └─ User quotas & usage metrics                                      │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 🔄 RESPONSE FLOW (Downstream → Upstream)                                 │
│                                                                          │
│     AI Service Response                                                  │
│            ↓                                                             │
│     Service Layer Processing                                            │
│     (save to DB, update counters)                                       │
│            ↓                                                             │
│     SendChatResponse DTO                                                │
│     {                                                                    │
│       message_id: uuid,                                                 │
│       chat: string,                                                      │
│       role: "model",                                                     │
│       created_at: timestamp,                                            │
│       session_id: uuid                                                  │
│     }                                                                    │
│            ↓                                                             │
│     Controller JSON Response (200 OK)                                   │
│            ↓                                                             │
│     HTTP Response ke Client                                             │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 🔍 DETAILED FLOW SEQUENCE

### **Request Lifecycle: SendChat**

```
1. HTTP REQUEST ARRIVES
   POST /chatbot/v1/send-chat
   Headers: { Authorization: Bearer <jwt_token> }
   Body: { chat_session_id: UUID, chat: "user question" }

2. MIDDLEWARE PROCESSING
   ├─ JwtMiddleware validates token
   ├─ Extract user_id dari claims
   └─ Attach user_id ke ctx.Locals("user_id")

3. CONTROLLER HANDLER (chatbot_controller.go:SendChat)
   ├─ c.chatbotService.SendChat(ctx, userId, request)
   └─ Delegate to SERVICE LAYER

4. SERVICE LAYER - AUTHENTICATION PHASE
   ├─ uow := cs.uowFactory.NewUnitOfWork(ctx)
   ├─ Fetch user subscriptions
   ├─ Validate active subscription exists
   ├─ Check plan.AiChatEnabled == true
   └─ Return error if not Pro Plan

5. SERVICE LAYER - TOKEN QUOTA ENFORCEMENT
   ├─ Fetch user record
   ├─ Check if AiDailyUsageLastReset < 24h ago
   ├─ If yes: reset AiDailyUsage counter
   ├─ Compare user.AiDailyUsage vs plan.AiChatDailyLimit
   └─ Return LimitExceededError if exceeded

6. SERVICE LAYER - SESSION & PERMISSION CHECK
   ├─ Query ChatSessionRepository with specifications:
   │  ├─ ByID{ID: request.ChatSessionId}
   │  └─ UserOwnedBy{UserID: userId}
   └─ Verify session exists AND belongs to user

7. SERVICE LAYER - CHAT HISTORY RETRIEVAL
   ├─ Fetch all ChatMessageRaw dari session
   ├─ Order by created_at ASC (preserve conversation flow)
   └─ Prepare for RAG decision step

8. SERVICE LAYER - EMBEDDING GENERATION
   ├─ Call embedding.GetGeminiEmbedding()
   ├─ Input: user question + task_type="RETRIEVAL_QUERY"
   ├─ Output: {embedding: {values: []float32}}
   └─ Used untuk semantic similarity search

9. SERVICE LAYER - RAG DECISION
   ├─ Prepare DecideUseRAGChatHistories slice
   ├─ Include system prompts dari constant
   ├─ Include recent user/model exchanges
   ├─ Call chatbot.DecideToUseRAGWithOllama()
   │  └─ Query Ollama: "Should we use knowledge base?"
   └─ Receive: boolean useRag

10. SERVICE LAYER - CONDITIONAL RETRIEVAL (IF useRag)
    ├─ Call NoteEmbeddingRepository.SearchSimilar()
    ├─ Input: embedding.Values, topK=5, userId
    ├─ Output: []NoteEmbedding{Document, Similarity Score}
    └─ Build context string dengan references

11. SERVICE LAYER - PROMPT CONSTRUCTION
    ├─ Build final prompt string:
    │  ├─ [Optional] Attach retrieved note references
    │  ├─ Include entire chat history (raw version)
    │  ├─ Append "User next question: {question}"
    │  └─ End dengan "Your answer ?"
    └─ Store dalam ChatMessageRaw entity

12. SERVICE LAYER - LLM CALL
    ├─ Prepare geminiReq slice dari chat histories
    ├─ Call chatbot.GetOllamaResponse(ctx, geminiReq)
    │  ├─ Serialize ke Ollama request format
    │  ├─ HTTP POST to localhost:11434/api/generate
    │  ├─ Stream/wait untuk response
    │  └─ Parse & return text
    └─ Receive: AI-generated response text

13. SERVICE LAYER - DATABASE PERSISTENCE
    ├─ uow.Begin(ctx) - Start transaction
    ├─ Create ChatMessage{role: user, chat: original_question}
    ├─ Create ChatMessageRaw{role: user, chat: full_prompt}
    ├─ Create ChatMessage{role: model, chat: ai_response}
    ├─ Create ChatMessageRaw{role: model, chat: ai_response}
    ├─ Update ChatSession.Title (jika first response)
    ├─ Update User.AiDailyUsage += token_count
    ├─ Update User.AiDailyUsageLastReset = now
    ├─ uow.Commit(ctx) - Transaction commit
    └─ If error: uow.Rollback()

14. SERVICE LAYER - RESPONSE CONSTRUCTION
    ├─ Create SendChatResponse DTO
    ├─ Include: message_id, chat, role, created_at, session_id
    └─ Return to CONTROLLER

15. CONTROLLER - RESPONSE FORMATTING
    ├─ Wrap response dalam SuccessResponse
    ├─ Set HTTP status 200 OK
    └─ ctx.JSON(response)

16. HTTP RESPONSE TO CLIENT
    └─ JSON: { status: "success", data: SendChatResponse }
```

---

## 📋 ENTITY & RELATIONSHIP MAP

```
┌──────────────────────────────────────────────────────────────┐
│                      User (root entity)                       │
│  ├─ id (PK)                                                  │
│  ├─ email                                                    │
│  ├─ AiDailyUsage (current tokens used today)                 │
│  ├─ AiDailyUsageLastReset (timestamp)                        │
│  └─ [FK] UserSubscription.user_id                            │
└─────────────────────┬──────────────────────────────────────────┘
                      │ 1:N relationship
                      ▼
┌──────────────────────────────────────────────────────────────┐
│              UserSubscription                                 │
│  ├─ id (PK)                                                  │
│  ├─ user_id (FK)                                             │
│  ├─ plan_id (FK)                                             │
│  ├─ status (active|inactive|cancelled)                       │
│  └─ [FK] SubscriptionPlan.plan_id                            │
└─────────────────────┬──────────────────────────────────────────┘
                      │ 1:N relationship
                      ▼
┌──────────────────────────────────────────────────────────────┐
│              SubscriptionPlan                                 │
│  ├─ id (PK)                                                  │
│  ├─ name ("Pro", "Free")                                     │
│  ├─ AiChatEnabled (boolean)                                  │
│  └─ AiChatDailyLimit (tokens per day)                        │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                   ChatSession                                 │
│  ├─ id (PK)                                                  │
│  ├─ user_id (FK) - ownership verification                    │
│  ├─ title                                                    │
│  ├─ created_at                                               │
│  └─ [1:N] ChatMessage.chat_session_id                        │
│     └─ [1:N] ChatMessageRaw.chat_session_id                  │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                  ChatMessage (Display Layer)                  │
│  ├─ id (PK)                                                  │
│  ├─ chat_session_id (FK)                                     │
│  ├─ chat (user-friendly text)                                │
│  ├─ role ("user" | "model")                                  │
│  ├─ created_at                                               │
│  └─ Purpose: Show in UI/API responses                        │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│              ChatMessageRaw (System Layer)                    │
│  ├─ id (PK)                                                  │
│  ├─ chat_session_id (FK)                                     │
│  ├─ chat (full prompt/response untuk LLM)                    │
│  ├─ role ("user" | "model")                                  │
│  ├─ created_at                                               │
│  └─ Purpose: Store actual prompts sent to LLM                │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                   NoteEmbedding (Vector Store)                │
│  ├─ id (PK)                                                  │
│  ├─ note_id (FK)                                             │
│  ├─ user_id (FK) - for multi-tenancy in search               │
│  ├─ embedding (pgvector column - float array)                │
│  ├─ document (original note text)                            │
│  ├─ created_at                                               │
│  └─ Purpose: Semantic search untuk RAG                       │
└──────────────────────────────────────────────────────────────┘
```

---

## 🔐 SECURITY & ISOLATION LAYERS

```
┌─────────────────────────────────────────────────────────┐
│ 1. JWT AUTHENTICATION LAYER                             │
│    ├─ Middleware: JwtMiddleware                          │
│    ├─ Extracts user_id dari JWT claims                  │
│    └─ Blocks unauthorized requests                      │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ 2. SUBSCRIPTION AUTHORIZATION                           │
│    ├─ Verify Pro Plan status                             │
│    ├─ Check AiChatEnabled feature flag                   │
│    └─ Block free tier users                              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ 3. QUOTA/RATE LIMITING                                  │
│    ├─ Daily token limit enforcement                      │
│    ├─ 24-hour rolling window reset                       │
│    └─ Real-time counter updates                          │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ 4. DATA OWNERSHIP VERIFICATION                          │
│    ├─ Session: UserOwnedBy{UserID: userId} spec         │
│    ├─ Note: user_id filter dalam SearchSimilar          │
│    └─ Prevents cross-user data leakage                  │
└─────────────────────────────────────────────────────────┘
```

---

## 🏗️ DEPENDENCY INJECTION CHAIN

```
cmd/rest/main.go
    └─ bootstrap.NewContainer(db, cfg)
        ├─ uowFactory := unitofwork.NewRepositoryFactory(db)
        │
        ├─ chatbotService := service.NewChatbotService(uowFactory)
        │   └─ Depends on: Repository Factory
        │
        └─ ChatbotController := controller.NewChatbotController(chatbotService)
            ├─ Depends on: IChatbotService
            └─ Injected via Constructor
```

---

## 🎯 KEY DECISION POINTS

| Decision Point         | Input             | Logic                            | Output            |
| ---------------------- | ----------------- | -------------------------------- | ----------------- |
| **Subscription Check** | user_id           | Query active subscription & plan | Error \| Continue |
| **Quota Enforcement**  | user, plan        | Compare AiDailyUsage vs limit    | Error \| Continue |
| **RAG Decision**       | chat history      | Query Ollama decision endpoint   | useRag: bool      |
| **Note Retrieval**     | embedding, userId | pgvector similarity search       | []NoteEmbedding   |
| **LLM Selection**      | fallback config   | Ollama available?                | Ollama \| Gemini  |

---

## 📊 TOKEN/DATA FLOW SUMMARY

```
User Input
    ↓
JWT Validation
    ↓
Subscription Check (Plan)
    ↓
Quota Check (Daily Limit)
    ↓
Session Ownership Check
    ↓
Generate Embedding (User Query)
    ↓
RAG Decision (Ollama)
    ↓
[IF RAG] Retrieve Similar Notes
    ↓
Build Prompt (with context)
    ↓
LLM Call (Ollama/Gemini)
    ↓
[TRANSACTION]
├─ Save User Message
├─ Save Raw Prompt
├─ Save Model Response
├─ Save Raw Response
├─ Update Session Title
└─ Update User Quota
    ↓
Return Response to Client
```

---

## 📁 FILE LOCATIONS REFERENCE

| Component                | File Location                                                                          |
| ------------------------ | -------------------------------------------------------------------------------------- |
| **Controller**           | [internal/controller/chatbot_controller.go](internal/controller/chatbot_controller.go) |
| **Service**              | [internal/service/chatbot_service.go](internal/service/chatbot_service.go)             |
| **Ollama Integration**   | [pkg/chatbot/ollama_chatbot.go](pkg/chatbot/ollama_chatbot.go)                         |
| **Gemini Integration**   | [pkg/chatbot/gemini_chatbot.go](pkg/chatbot/gemini_chatbot.go)                         |
| **Embedding Service**    | [pkg/embedding/](pkg/embedding/)                                                       |
| **Repository Contracts** | [internal/repository/contract/](internal/repository/contract/)                         |
| **Unit of Work**         | [internal/repository/unitofwork/](internal/repository/unitofwork/)                     |
| **Container/DI**         | [internal/bootstrap/container.go](internal/bootstrap/container.go)                     |
| **Config Loading**       | [internal/config/config.go](internal/config/config.go)                                 |
| **Database**             | [pkg/database/](pkg/database/)                                                         |

---

## ⚙️ CONFIGURATION & ENVIRONMENT

```env
# API Keys (Required for external services)
GOOGLE_GEMINI_API_KEY=<api_key_for_embedding_and_fallback>

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/notefiber

# Ollama Server (Local LLM)
OLLAMA_ENDPOINT=http://localhost:11434

# Models
OLLAMA_MODEL=llama2|mistral|neural-chat
GEMINI_MODEL=gemini-1.5-flash
```

---

## 🚀 DEPLOYMENT CONSIDERATIONS

1. **Dependency Order:**

   - PostgreSQL must be running
   - Ollama server should be available (else fallback to Gemini)
   - Google API credentials must be configured

2. **Vector Search:**

   - PostgreSQL pgvector extension required
   - Note embeddings must be pre-computed

3. **Quotas:**

   - Per-user daily limits tracked in-DB
   - Reset logic tied to user record timestamps

4. **Scalability:**
   - Service layer handles transactional integrity
   - Repository layer abstracts DB operations
   - AI calls are blocking (consider async later)

---

**Generated:** 28 December 2025  
**Framework:** Go + Fiber  
**Architecture Pattern:** Layered Architecture + Repository Pattern + Unit of Work
