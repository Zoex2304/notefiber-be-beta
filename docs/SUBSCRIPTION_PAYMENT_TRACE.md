# 💳 Subscription System & Payment Gateway - Upstream to Downstream Trace

## Dokumentasi Path Alur Data (Trace Deep)
**Tanggal:** 28 December 2025  
**Aplikasi:** Note Fiber Backend - Subscription & Payment Integration

---

## 📊 UPSTREAM → DOWNSTREAM ARCHITECTURE

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        HTTP REQUEST (Entry Points)                       │
│   1. GET /payment/plans              (Public - List all plans)           │
│   2. GET /payment/summary?plan_id    (Public - Order preview)            │
│   3. POST /payment/checkout          (Protected - Initiate payment)      │
│   4. POST /payment/midtrans/webhook  (Midtrans callback - Payment status)│
│   5. GET /payment/status             (Protected - Subscription status)   │
│   6. POST /payment/cancel            (Protected - Cancel subscription)   │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 1️⃣  CONTROLLER LAYER (HTTP Handler & Request Validation)               │
│     📄 internal/controller/payment_controller.go                        │
│                                                                          │
│     Endpoints:                                                           │
│     ├─ RegisterRoutes() - Define payment routes                         │
│     │                                                                    │
│     ├─ GetPlans(ctx *fiber.Ctx)                                         │
│     │  ├─ No authentication required                                    │
│     │  ├─ Return list of available subscription plans                   │
│     │  └─ Includes: name, price, features, description                 │
│     │                                                                    │
│     ├─ GetOrderSummary(ctx *fiber.Ctx)                                  │
│     │  ├─ Query param: plan_id (required)                               │
│     │  ├─ Calculate: subtotal, tax, total                               │
│     │  └─ No authentication needed (preview feature)                    │
│     │                                                                    │
│     ├─ Checkout(ctx *fiber.Ctx)                                         │
│     │  ├─ Requires JWT authentication                                   │
│     │  ├─ Parse request: plan_id, billing address, email                │
│     │  ├─ Validate request body                                         │
│     │  ├─ Extract user_id dari JWT                                      │
│     │  └─ Delegate ke PaymentService.CreateSubscription()              │
│     │                                                                    │
│     ├─ Webhook(ctx *fiber.Ctx)                                          │
│     │  ├─ Receive Midtrans callback (no auth)                           │
│     │  ├─ Parse webhook payload                                         │
│     │  └─ Delegate ke PaymentService.HandleNotification()              │
│     │                                                                    │
│     ├─ GetStatus(ctx *fiber.Ctx)                                        │
│     │  ├─ Requires JWT authentication                                   │
│     │  ├─ Extract user_id dari JWT                                      │
│     │  └─ Return subscription status (active/inactive/free)             │
│     │                                                                    │
│     └─ CancelSubscription(ctx *fiber.Ctx)                               │
│        ├─ Requires JWT authentication                                   │
│        ├─ Extract user_id dari JWT                                      │
│        └─ Delegate ke PaymentService.CancelSubscription()              │
│                                                                          │
│     Dependency: IPaymentService                                         │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 2️⃣  SERVICE LAYER (Business Logic & Payment Orchestration)             │
│     📄 internal/service/payment_service.go                              │
│                                                                          │
│     A. GetPlans() Flow:                                                  │
│     ├─ Query all SubscriptionPlan dari database                         │
│     ├─ Transform to DTO (name, price, features)                         │
│     └─ Return list of plans dengan feature descriptions                 │
│                                                                          │
│     B. GetOrderSummary() Flow:                                           │
│     ├─ Fetch plan by planId                                             │
│     ├─ Calculate: Subtotal = plan.Price                                 │
│     ├─ Calculate: Tax = Subtotal * plan.TaxRate                         │
│     ├─ Calculate: Total = Subtotal + Tax                                │
│     └─ Return OrderSummaryResponse dengan breakdown                     │
│                                                                          │
│     C. CreateSubscription() Flow (MOST COMPLEX):                         │
│     ├─ 🔐 AUTHENTICATION & VALIDATION                                   │
│     │  ├─ Verify user exists                                            │
│     │  ├─ Verify plan exists                                            │
│     │  └─ Validate billing address data                                 │
│     │                                                                    │
│     ├─ 💾 PERSIST BILLING ADDRESS                                       │
│     │  ├─ Create BillingAddress entity                                  │
│     │  └─ Store dalam database                                          │
│     │                                                                    │
│     ├─ 💳 CREATE SUBSCRIPTION RECORD                                    │
│     │  ├─ Generate new UUID untuk subscription                          │
│     │  ├─ Set status = "inactive" (menunggu payment)                    │
│     │  ├─ Set paymentStatus = "pending"                                 │
│     │  ├─ Calculate period dates (monthly/yearly)                       │
│     │  └─ Store dalam database (within transaction)                     │
│     │                                                                    │
│     ├─ 🌐 INITIATE PAYMENT GATEWAY (Midtrans)                            │
│     │  ├─ Set Midtrans environment (Sandbox/Production)                 │
│     │  ├─ Prepare snap.Request dengan:                                  │
│     │  │  ├─ OrderID = subscription_id                                  │
│     │  │  ├─ GrossAmount = (Price * (1 + TaxRate))                      │
│     │  │  ├─ Customer details (name, email, address)                    │
│     │  │  ├─ Item details (plan name, price, qty)                       │
│     │  │  ├─ Payment methods (credit card, bank transfer, etc)          │
│     │  │  └─ Callbacks untuk success/finish flows                       │
│     │  │                                                                 │
│     │  └─ Call snap.CreateTransaction()                                 │
│     │     └─ Returns: snapToken, redirectURL                            │
│     │                                                                    │
│     └─ 📤 RETURN CHECKOUT RESPONSE                                      │
│        ├─ subscription_id (untuk reference)                             │
│        └─ snap_redirect_url (redirect ke payment page)                  │
│                                                                          │
│     D. HandleNotification() Flow (Webhook from Midtrans):               │
│     ├─ 🔐 VALIDATE WEBHOOK SIGNATURE                                    │
│     │  ├─ Verify signature using SHA512(OrderId + Payload)             │
│     │  └─ Ensure authenticity dari Midtrans                             │
│     │                                                                    │
│     ├─ 📥 PARSE NOTIFICATION DATA                                       │
│     │  ├─ Extract OrderId (= subscription_id)                           │
│     │  ├─ Extract TransactionStatus (capture/settlement/deny/etc)      │
│     │  ├─ Extract PaymentStatus (success/failed/pending)               │
│     │  └─ Extract FraudStatus                                           │
│     │                                                                    │
│     ├─ 🔄 DETERMINE NEW SUBSCRIPTION STATUS                             │
│     │  ├─ IF TransactionStatus = "capture" or "settlement"              │
│     │  │  ├─ Set SubscriptionStatus = "active"                          │
│     │  │  └─ Set PaymentStatus = "success"                              │
│     │  │                                                                 │
│     │  ├─ ELSE IF TransactionStatus = "deny/cancel/expire"              │
│     │  │  ├─ Set SubscriptionStatus = "inactive"                        │
│     │  │  └─ Set PaymentStatus = "failed"                               │
│     │  │                                                                 │
│     │  └─ ELSE (pending/other)                                          │
│     │     └─ Skip update, return OK                                     │
│     │                                                                    │
│     ├─ 💾 UPDATE DATABASE                                               │
│     │  ├─ Begin transaction                                             │
│     │  ├─ Update UserSubscription status                                │
│     │  ├─ Commit transaction                                            │
│     │  └─ Rollback on error                                             │
│     │                                                                    │
│     └─ 📊 LOGGING                                                       │
│        ├─ Log transaction details untuk audit trail                     │
│        └─ Log state transitions (pending → active/inactive)             │
│                                                                          │
│     E. GetSubscriptionStatus() Flow:                                     │
│     ├─ Query user's subscriptions (all records)                         │
│     ├─ Determine "active" subscription berdasarkan criteria:            │
│     │  ├─ Status must be "active"                                       │
│     │  ├─ Period end date must be in future                             │
│     │  └─ Priority: active > payment_succeeded > inactive               │
│     │                                                                    │
│     ├─ IF active subscription found:                                    │
│     │  ├─ Fetch corresponding plan                                      │
│     │  ├─ Return: plan name, status, limits, daily quotas              │
│     │  └─ Features enabled: AI chat, semantic search, limits            │
│     │                                                                    │
│     └─ ELSE (no active subscription):                                   │
│        └─ Return: Free Plan defaults (3 notebooks, 10 notes, no AI)     │
│                                                                          │
│     F. CancelSubscription() Flow:                                        │
│     ├─ Find user's active subscription                                  │
│     ├─ Set status = "canceled"                                          │
│     ├─ Update CurrentPeriodEnd = now()                                  │
│     ├─ Persist to database                                              │
│     └─ Return success                                                   │
│                                                                          │
│     Dependency: SubscriptionRepository, BillingRepository, Midtrans SDK │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 3️⃣  REPOSITORY LAYER (Data Access & Domain)                             │
│     📄 internal/repository/                                             │
│                                                                          │
│     A. Subscription Domain Repositories:                                │
│        ├─ SubscriptionRepository                                        │
│        │  ├─ CreatePlan() - Create subscription plan                    │
│        │  ├─ UpdatePlan() - Update plan details                         │
│        │  ├─ DeletePlan() - Delete plan                                 │
│        │  ├─ FindOnePlan() - Query plan by ID/slug                      │
│        │  ├─ FindAllPlans() - List all active plans                     │
│        │  ├─ CreateSubscription() - Create user subscription            │
│        │  ├─ UpdateSubscription() - Update subscription status          │
│        │  ├─ DeleteSubscription() - Soft/hard delete                    │
│        │  ├─ FindOneSubscription() - Query by ID/user                   │
│        │  └─ FindAllSubscriptions() - List user's subscriptions         │
│        │                                                                 │
│     B. Billing Domain Repositories:                                     │
│        └─ BillingRepository                                             │
│           ├─ Create() - Save billing address                            │
│           ├─ FindOne() - Query billing address                          │
│           └─ Update() - Update address details                          │
│                                                                          │
│     C. User Repository (used untuk validation):                         │
│        └─ UserRepository                                                │
│           └─ FindOne() - Verify user exists                             │
│                                                                          │
│     Unit of Work Pattern:                                               │
│        ├─ Transactional consistency (billing + subscription)            │
│        ├─ Either both succeed or both rollback                          │
│        └─ Ensures data integrity during checkout                        │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 4️⃣  PAYMENT GATEWAY INTEGRATION LAYER (Midtrans SDK)                    │
│     📄 github.com/midtrans/midtrans-go                                  │
│                                                                          │
│     Midtrans Snap API Integration:                                      │
│     ├─ Server Key: Authentication untuk backend                         │
│     ├─ Client Key: Untuk frontend token generation                      │
│     ├─ Environment: Sandbox (testing) atau Production (live)            │
│     │                                                                    │
│     │ CreateTransaction Flow:                                           │
│     ├─ Prepare snap.Request dengan payment details                      │
│     ├─ Include customer information                                     │
│     ├─ Set enabled payment methods                                      │
│     ├─ Call snap.CreateTransaction()                                    │
│     └─ Receive snapToken & redirectURL                                  │
│                                                                          │
│     Webhook Notification Handling:                                      │
│     ├─ Receive POST dari Midtrans server                                │
│     ├─ Extract order_id (subscription_id)                               │
│     ├─ Extract transaction_status (payment outcome)                     │
│     ├─ Validate signature untuk authenticity                            │
│     └─ Update subscription status accordingly                           │
│                                                                          │
│     Supported Payment Methods:                                          │
│     ├─ Credit Card (Visa, Mastercard, JCB)                              │
│     ├─ Debit Card (various banks)                                       │
│     ├─ Bank Transfer (virtual account, ATM)                             │
│     ├─ E-wallet (GCash, OVO, Dana, LINKAJA)                             │
│     ├─ Buy Now Pay Later (Akulaku, kredivo)                             │
│     └─ Convenience Store (Indomaret, Alfamart)                          │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 5️⃣  EXTERNAL PAYMENT SERVICE (Midtrans & PostgreSQL)                    │
│                                                                          │
│     🏦 Midtrans Payment Gateway (Verifone subsidiary)                    │
│     ├─ Endpoint (Sandbox): https://app.sandbox.midtrans.com/snap/v1/... │
│     ├─ Endpoint (Prod): https://app.midtrans.com/snap/v1/...           │
│     ├─ Authentication: Server Key (SHA512 signature)                    │
│     ├─ Purpose: Process payments, handle multiple payment methods       │
│     └─ Webhook Callback: POST to /payment/midtrans/notification        │
│                                                                          │
│     🗄️  PostgreSQL Database Tables:                                     │
│     ├─ subscription_plans                                               │
│     │  ├─ id (PK)                                                       │
│     │  ├─ name, slug, description, tagline                              │
│     │  ├─ price, tax_rate, billing_period                               │
│     │  ├─ max_notebooks, max_notes_per_notebook                         │
│     │  ├─ semantic_search_enabled, ai_chat_enabled                      │
│     │  ├─ ai_chat_daily_limit, semantic_search_daily_limit              │
│     │  ├─ is_most_popular, is_active, sort_order                        │
│     │  └─ created_at, updated_at                                        │
│     │                                                                    │
│     ├─ user_subscriptions                                               │
│     │  ├─ id (PK)                                                       │
│     │  ├─ user_id (FK) - ownership                                      │
│     │  ├─ plan_id (FK) - linked plan                                    │
│     │  ├─ billing_address_id (FK)                                       │
│     │  ├─ status (active|inactive|canceled)                             │
│     │  ├─ payment_status (pending|success|failed)                       │
│     │  ├─ current_period_start, current_period_end                      │
│     │  ├─ created_at, updated_at                                        │
│     │  └─ auto_renew (boolean flag)                                     │
│     │                                                                    │
│     └─ billing_addresses                                                │
│        ├─ id (PK)                                                       │
│        ├─ user_id (FK)                                                  │
│        ├─ first_name, last_name, email, phone                           │
│        ├─ address_line1, address_line2, city, state                     │
│        ├─ postal_code, country                                          │
│        ├─ is_default                                                    │
│        └─ created_at, updated_at                                        │
└──────────────────────────────┬──────────────────────────────────────────┘
                               │
                    ▼          │
┌─────────────────────────────────────────────────────────────────────────┐
│ 🔄 RESPONSE FLOW (Downstream → Upstream)                                 │
│                                                                          │
│ ┌─────── PAYMENT SUCCESS FLOW ──────────┐                               │
│ │                                       │                               │
│ │ 1. Client submits checkout form       │                               │
│ │ 2. Controller receives & validates    │                               │
│ │ 3. Service creates subscription record│                               │
│ │ 4. Service calls Midtrans API         │                               │
│ │ 5. Return snapToken to client         │                               │
│ │ 6. Client redirects to Midtrans page  │                               │
│ │ 7. User completes payment             │                               │
│ │ 8. Midtrans processes payment         │                               │
│ │ 9. Midtrans sends webhook callback    │                               │
│ │10. Service updates subscription status│                               │
│ │11. User sees success & redirected     │                               │
│ │                                       │                               │
│ └───────────────────────────────────────┘                               │
│                                                                          │
│ Response DTOs:                                                           │
│ ├─ CheckoutResponse:                                                    │
│ │  ├─ subscription_id: UUID                                             │
│ │  └─ snap_redirect_url: string                                         │
│ │                                                                        │
│ ├─ SubscriptionStatusResponse:                                          │
│ │  ├─ subscription_id: UUID                                             │
│ │  ├─ plan_name: string                                                 │
│ │  ├─ status: "active"|"inactive"|"canceled"                            │
│ │  ├─ is_active: boolean                                                │
│ │  ├─ current_period_end: timestamp                                     │
│ │  ├─ ai_chat_daily_limit: int                                          │
│ │  ├─ semantic_search_daily_limit: int                                  │
│ │  └─ features: { aiChat, semanticSearch, maxNotebooks, ... }          │
│ │                                                                        │
│ └─ OrderSummaryResponse:                                                │
│    ├─ plan_name: string                                                 │
│    ├─ billing_period: "month"|"year"                                    │
│    ├─ price_per_unit: string (e.g., "$9/month")                         │
│    ├─ subtotal: float64                                                 │
│    ├─ tax: float64                                                      │
│    ├─ total: float64                                                    │
│    └─ currency: "USD"|"IDR"                                             │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 🔍 DETAILED FLOW SEQUENCE

### **Request Lifecycle 1: GET Plans (Discovery)**

```
1. HTTP REQUEST ARRIVES
   GET /payment/plans
   (No authentication required)

2. CONTROLLER HANDLER (payment_controller.go:GetPlans)
   ├─ c.service.GetPlans(ctx.Context())
   └─ Delegate to SERVICE LAYER

3. SERVICE LAYER - FETCH PLANS
   ├─ uow := s.uowFactory.NewUnitOfWork(ctx)
   ├─ plans := uow.SubscriptionRepository().FindAllPlans(ctx)
   │  └─ Query database: SELECT * FROM subscription_plans WHERE is_active = true
   │
   └─ Transform to DTO:
      ├─ For each plan:
      │  ├─ Build features list
      │  ├─ Include: AI Chat, Semantic Search (if enabled)
      │  └─ Add: Storage limits, pricing info
      └─ Return []*PlanResponse

4. REPOSITORY LAYER
   ├─ Execute GORM query
   ├─ Scan results into []*entity.SubscriptionPlan
   └─ Return to service

5. CONTROLLER - RESPONSE FORMATTING
   ├─ Wrap in SuccessResponse
   ├─ Set HTTP status 200 OK
   └─ ctx.JSON(response)

6. HTTP RESPONSE TO CLIENT
   ```json
   {
     "success": true,
     "code": 200,
     "message": "Success fetching plans",
     "data": [
       {
         "id": "uuid-1",
         "name": "Pro Plan",
         "slug": "pro",
         "price": 50000,
         "description": "Unlimited AI features",
         "features": ["AI Chat", "Semantic Search"]
       },
       ...
     ]
   }
   ```
```

---

### **Request Lifecycle 2: GET Order Summary (Preview)**

```
1. HTTP REQUEST ARRIVES
   GET /payment/summary?plan_id=uuid-1
   (No authentication required - preview feature)

2. CONTROLLER HANDLER (payment_controller.go:GetOrderSummary)
   ├─ Extract query param: plan_id
   ├─ Validate plan_id format
   └─ c.service.GetOrderSummary(ctx.Context(), planId)

3. SERVICE LAYER - CALCULATE ORDER
   ├─ uow := s.uowFactory.NewUnitOfWork(ctx)
   ├─ plan := uow.SubscriptionRepository().FindOnePlan(ctx, ByID{ID: planId})
   ├─ Validate plan exists
   │
   └─ Calculate:
      ├─ subtotal = plan.Price (e.g., 50000)
      ├─ tax = subtotal * plan.TaxRate (e.g., 50000 * 0.11 = 5500)
      ├─ total = subtotal + tax (e.g., 55500)
      └─ billingPeriod = "month" or "year" dari plan.BillingPeriod

4. CONTROLLER - RESPONSE FORMATTING
   ├─ Create OrderSummaryResponse
   └─ Return JSON dengan breakdown

5. HTTP RESPONSE TO CLIENT
   ```json
   {
     "success": true,
     "data": {
       "plan_name": "Pro Plan",
       "billing_period": "month",
       "price_per_unit": "$50.00/month",
       "subtotal": 50000,
       "tax": 5500,
       "total": 55500,
       "currency": "USD"
     }
   }
   ```
```

---

### **Request Lifecycle 3: POST Checkout (Most Complex - Payment Initiation)**

```
1. HTTP REQUEST ARRIVES
   POST /payment/checkout
   Headers: { Authorization: Bearer <jwt_token> }
   Body: {
     "plan_id": "uuid-1",
     "first_name": "John",
     "last_name": "Doe",
     "email": "john@example.com",
     "phone": "+62812345678",
     "address_line1": "Jl. Sudirman No. 1",
     "address_line2": "Apt. 2B",
     "city": "Jakarta",
     "state": "DKI Jakarta",
     "postal_code": "12190",
     "country": "ID"
   }

2. MIDDLEWARE PROCESSING
   ├─ JwtMiddleware validates token
   ├─ Extract user_id dari claims
   └─ Attach user_id ke ctx.Locals("user_id")

3. CONTROLLER HANDLER (payment_controller.go:Checkout)
   ├─ Parse request body ke dto.CheckoutRequest
   ├─ Validate request (all required fields present)
   ├─ Extract user_id dari JWT
   └─ c.service.CreateSubscription(ctx, userId, &request)

4. SERVICE LAYER - PHASE 1: VALIDATION
   ├─ uow := s.uowFactory.NewUnitOfWork(ctx)
   ├─ Fetch user by ID (verify exists)
   ├─ Fetch plan by ID (verify exists)
   └─ Return error if validation fails

5. SERVICE LAYER - PHASE 2: BILLING ADDRESS
   ├─ Create BillingAddress entity:
   │  ├─ id = uuid.New()
   │  ├─ user_id = userId
   │  ├─ Populate dari request (name, address, city, etc)
   │  └─ is_default = true
   └─ NOT saved yet (part of transaction)

6. SERVICE LAYER - PHASE 3: SUBSCRIPTION RECORD
   ├─ Create UserSubscription entity:
   │  ├─ id = uuid.New()
   │  ├─ user_id = userId
   │  ├─ plan_id = request.PlanId
   │  ├─ billing_address_id = billingAddressId
   │  ├─ status = "inactive" (waiting for payment)
   │  ├─ payment_status = "pending"
   │  ├─ current_period_start = time.Now()
   │  ├─ IF plan.BillingPeriod == "monthly":
   │  │  └─ current_period_end = Now().AddDate(0, 1, 0)
   │  └─ IF plan.BillingPeriod == "yearly":
   │     └─ current_period_end = Now().AddDate(1, 0, 0)
   └─ NOT saved yet (part of transaction)

7. SERVICE LAYER - PHASE 4: DATABASE TRANSACTION
   ├─ uow.Begin(ctx)
   ├─ Save BillingAddress:
   │  └─ uow.BillingRepository().Create(ctx, billingAddr)
   ├─ Save UserSubscription:
   │  └─ uow.SubscriptionRepository().CreateSubscription(ctx, sub)
   ├─ uow.Commit()
   └─ Rollback on error

8. SERVICE LAYER - PHASE 5: MIDTRANS INTEGRATION
   ├─ Load Midtrans configuration:
   │  ├─ serverKey = os.Getenv("MIDTRANS_SERVER_KEY")
   │  ├─ environment = Sandbox or Production
   │  └─ client := snap.Client.New(serverKey, env)
   │
   ├─ Prepare snap.Request:
   │  ├─ TransactionDetails:
   │  │  ├─ OrderID = subscriptionId.String() (unique)
   │  │  └─ GrossAmount = int64(plan.Price + (plan.Price * plan.TaxRate))
   │  │
   │  ├─ CreditCard:
   │  │  └─ Secure = true
   │  │
   │  ├─ CustomerDetail:
   │  │  ├─ FName, LName, Email, Phone
   │  │  └─ BillingAddress (structured format untuk Midtrans)
   │  │
   │  ├─ Items:
   │  │  └─ []{ID: plan.Id, Name: plan.Name, Price: plan.Price, Qty: 1}
   │  │
   │  ├─ EnabledPayments:
   │  │  └─ AllSnapPaymentType (credit card, bank transfer, e-wallet, etc)
   │  │
   │  └─ Callbacks:
   │     └─ Finish: frontendURL/app?payment=success
   │
   └─ Call snap.CreateTransaction(snapReq)
      └─ Returns: snapToken, redirectURL

9. SERVICE LAYER - RETURN RESPONSE
   ├─ Create CheckoutResponse:
   │  ├─ subscription_id = subscriptionId
   │  └─ snap_redirect_url = redirectURL
   └─ Return to controller

10. CONTROLLER - RESPONSE FORMATTING
    ├─ Wrap in SuccessResponse
    └─ Return JSON

11. HTTP RESPONSE TO CLIENT
    ```json
    {
      "success": true,
      "code": 200,
      "message": "Subscription created",
      "data": {
        "subscription_id": "550e8400-e29b-41d4-a716-446655440000",
        "snap_redirect_url": "https://app.sandbox.midtrans.com/snap/v1/..."
      }
    }
    ```

12. CLIENT REDIRECT
    ├─ JavaScript redirect ke snap_redirect_url
    └─ User lands on Midtrans payment page

13. USER PAYMENT
    ├─ User chooses payment method
    ├─ Completes payment process
    └─ Payment gateway processes transaction

14. MIDTRANS WEBHOOK CALLBACK
    ├─ After payment result known
    ├─ Midtrans sends POST ke /payment/midtrans/notification
    └─ See webhook flow below
```

---

### **Request Lifecycle 4: POST Webhook (Asynchronous - Payment Result)**

```
1. MIDTRANS SENDS WEBHOOK
   POST /payment/midtrans/notification
   Headers: { Content-Type: application/json }
   Body: {
     "transaction_id": "123456789",
     "order_id": "550e8400-e29b-41d4-a716-446655440000",  // subscription_id
     "transaction_status": "settlement",  // or capture, deny, cancel, expire, pending
     "payment_type": "credit_card",
     "gross_amount": "55500",
     "signature_key": "sha512_hash_here",
     "fraud_status": "accept"
   }

2. CONTROLLER HANDLER (payment_controller.go:Webhook)
   ├─ Parse webhook body
   └─ c.service.HandleNotification(ctx, &webhookReq)

3. SERVICE LAYER - PHASE 1: SIGNATURE VALIDATION
   ├─ Verify webhook authenticity
   ├─ Reconstruct signature:
   │  ├─ sig_string = order_id + transaction_status + gross_amount + server_key
   │  ├─ calculated_sig = SHA512(sig_string)
   │  └─ Compare dengan request.signature_key
   └─ Return error if signature invalid (prevent spoofing)

4. SERVICE LAYER - PHASE 2: EXTRACT NOTIFICATION DATA
   ├─ subId = uuid.Parse(order_id)
   ├─ transactionStatus = req.TransactionStatus
   └─ fraudStatus = req.FraudStatus

5. SERVICE LAYER - PHASE 3: DETERMINE NEW STATUS
   ├─ IF transactionStatus == "capture" OR "settlement":
   │  ├─ newStatus = SubscriptionStatusActive
   │  └─ newPaymentStatus = PaymentStatusPaid
   │
   ├─ ELSE IF transactionStatus == "deny" OR "cancel" OR "expire":
   │  ├─ newStatus = SubscriptionStatusInactive
   │  └─ newPaymentStatus = PaymentStatusFailed
   │
   └─ ELSE IF transactionStatus == "pending":
      └─ Skip update, return OK (payment still pending)

6. SERVICE LAYER - PHASE 4: FETCH SUBSCRIPTION
   ├─ uow := s.uowFactory.NewUnitOfWork(ctx)
   ├─ sub := uow.SubscriptionRepository().FindOneSubscription(ctx, ByID{ID: subId})
   ├─ Verify subscription exists
   └─ Check current status (avoid duplicate updates)

7. SERVICE LAYER - PHASE 5: UPDATE SUBSCRIPTION
   ├─ IF status changed:
   │  ├─ uow.Begin(ctx)
   │  ├─ sub.Status = newStatus
   │  ├─ sub.PaymentStatus = newPaymentStatus
   │  ├─ sub.UpdatedAt = time.Now()
   │  ├─ uow.SubscriptionRepository().UpdateSubscription(ctx, sub)
   │  ├─ uow.Commit()
   │  └─ Rollback on error
   │
   └─ ELSE:
      └─ Skip update (already updated)

8. SERVICE LAYER - LOGGING & AUDIT
   ├─ Log webhook received
   ├─ Log status transitions
   ├─ Log any errors
   └─ Store untuk audit trail

9. CONTROLLER - RESPONSE
   ├─ Return HTTP 200 OK
   └─ Confirm webhook received

10. HTTP RESPONSE TO MIDTRANS
    ```json
    {
      "success": true,
      "code": 200,
      "message": "Webhook processed"
    }
    ```
    (Important: Must return 200 OK to prevent Midtrans retries)

11. DATABASE STATE
    ├─ IF payment success:
    │  └─ UserSubscription.status = "active"
    │     └─ User now has full plan access
    │
    └─ IF payment failed:
       └─ UserSubscription.status = "inactive"
          └─ User remains on free plan
```

---

### **Request Lifecycle 5: GET Subscription Status (Verification)**

```
1. HTTP REQUEST ARRIVES
   GET /payment/status
   Headers: { Authorization: Bearer <jwt_token> }

2. MIDDLEWARE PROCESSING
   ├─ JwtMiddleware validates token
   ├─ Extract user_id dari claims
   └─ Attach user_id ke ctx.Locals("user_id")

3. CONTROLLER HANDLER (payment_controller.go:GetStatus)
   ├─ Extract user_id dari JWT
   └─ c.service.GetSubscriptionStatus(ctx.Context(), userId)

4. SERVICE LAYER - FETCH SUBSCRIPTIONS
   ├─ uow := s.uowFactory.NewUnitOfWork(ctx)
   ├─ subs := uow.SubscriptionRepository().FindAllSubscriptions(ctx,
   │                                        UserOwnedBy{UserID: userId})
   │
   └─ Query hasil: []*UserSubscription (all user's subscriptions)

5. SERVICE LAYER - DETERMINE ACTIVE SUBSCRIPTION
   ├─ Iterate through subscriptions dengan priority:
   │
   ├─ Priority 1: Active status + valid period
   │  └─ IF sub.Status == "active" AND sub.CurrentPeriodEnd.After(now):
   │     └─ activeSub = sub
   │
   ├─ Priority 2: Payment succeeded + valid period
   │  └─ IF no active found AND sub.PaymentStatus == "success":
   │     └─ activeSub = sub
   │
   └─ Priority 3: None found
      └─ Return free plan defaults

6. SERVICE LAYER - BUILD RESPONSE
   ├─ IF activeSub found:
   │  ├─ Fetch plan: uow.SubscriptionRepository().FindOnePlan(ctx, ByID{ID: activeSub.PlanId})
   │  └─ Create SubscriptionStatusResponse:
   │     ├─ subscription_id = activeSub.Id
   │     ├─ plan_name = plan.Name
   │     ├─ status = string(activeSub.Status)
   │     ├─ is_active = true
   │     ├─ current_period_end = activeSub.CurrentPeriodEnd
   │     ├─ ai_chat_daily_limit = plan.AiChatDailyLimit
   │     ├─ semantic_search_daily_limit = plan.SemanticSearchDailyLimit
   │     └─ features = {aiChat: true, semanticSearch: true, ...}
   │
   └─ ELSE (no active subscription):
      └─ Create SubscriptionStatusResponse (FREE PLAN):
         ├─ plan_name = "Free Plan"
         ├─ status = "inactive"
         ├─ is_active = false
         ├─ ai_chat_daily_limit = 0
         ├─ semantic_search_daily_limit = 0
         └─ features = {aiChat: false, semanticSearch: false, maxNotebooks: 3, ...}

7. CONTROLLER - RESPONSE FORMATTING
   ├─ Wrap in SuccessResponse
   └─ Return JSON

8. HTTP RESPONSE TO CLIENT
   ```json
   {
     "success": true,
     "data": {
       "subscription_id": "550e8400-e29b-41d4-a716-446655440000",
       "plan_name": "Pro Plan",
       "status": "active",
       "is_active": true,
       "current_period_end": "2026-01-28T10:30:00Z",
       "ai_chat_daily_limit": 50,
       "semantic_search_daily_limit": 100,
       "features": {
         "ai_chat": true,
         "semantic_search": true,
         "max_notebooks": 10,
         "max_notes_per_notebook": 100
       }
     }
   }
   ```
```

---

### **Request Lifecycle 6: POST Cancel Subscription**

```
1. HTTP REQUEST ARRIVES
   POST /payment/cancel
   Headers: { Authorization: Bearer <jwt_token> }

2. MIDDLEWARE & CONTROLLER
   ├─ Validate JWT
   ├─ Extract user_id
   └─ c.service.CancelSubscription(ctx.Context(), userId)

3. SERVICE LAYER - FIND SUBSCRIPTION
   ├─ uow := s.uowFactory.NewUnitOfWork(ctx)
   ├─ Find user's active subscription
   ├─ Verify ownership (subscription belongs to user)
   └─ Return error if not found

4. SERVICE LAYER - UPDATE STATUS
   ├─ sub.Status = SubscriptionStatusCanceled
   ├─ sub.CurrentPeriodEnd = time.Now() (immediately cancel)
   ├─ uow.SubscriptionRepository().UpdateSubscription(ctx, sub)
   └─ Return success

5. HTTP RESPONSE TO CLIENT
   ```json
   {
     "success": true,
     "code": 200,
     "message": "Subscription canceled successfully"
   }
   ```
```

---

## 📋 ENTITY & RELATIONSHIP MAP

```
┌──────────────────────────────────────────────────────────────┐
│                      User (root entity)                       │
│  ├─ id (PK)                                                  │
│  ├─ email                                                    │
│  └─ [1:N] UserSubscription.user_id                           │
│     └─ [1:N] BillingAddress.user_id                          │
└─────────────────────┬──────────────────────────────────────────┘
                      │ 1:N relationship
                      ▼
┌──────────────────────────────────────────────────────────────┐
│              UserSubscription (Subscription Record)           │
│  ├─ id (PK)                                                  │
│  ├─ user_id (FK) - ownership verification                    │
│  ├─ plan_id (FK) - linked plan                               │
│  ├─ billing_address_id (FK)                                  │
│  ├─ status (active|inactive|canceled)                        │
│  ├─ payment_status (pending|success|failed)                  │
│  ├─ current_period_start (timestamp)                         │
│  ├─ current_period_end (timestamp)                           │
│  ├─ auto_renew (boolean - future feature)                    │
│  ├─ created_at, updated_at                                   │
│  └─ [1:N] RelatedPayments (future audit trail)               │
└─────────────────────┬──────────────────────────────────────────┘
                      │ N:1 relationship
                      ▼
┌──────────────────────────────────────────────────────────────┐
│              SubscriptionPlan (Plan Definition)               │
│  ├─ id (PK)                                                  │
│  ├─ name (e.g., "Pro Plan")                                  │
│  ├─ slug (e.g., "pro" - unique)                              │
│  ├─ description, tagline                                     │
│  ├─ price, tax_rate, billing_period                          │
│  ├─ max_notebooks, max_notes_per_notebook                    │
│  ├─ semantic_search_enabled, ai_chat_enabled                 │
│  ├─ ai_chat_daily_limit, semantic_search_daily_limit         │
│  ├─ is_most_popular, is_active, sort_order                   │
│  ├─ created_at, updated_at                                   │
│  └─ [1:N] UserSubscription.plan_id                           │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│              BillingAddress (Billing Info)                    │
│  ├─ id (PK)                                                  │
│  ├─ user_id (FK)                                             │
│  ├─ first_name, last_name, email, phone                      │
│  ├─ address_line1, address_line2                             │
│  ├─ city, state, postal_code, country                        │
│  ├─ is_default (boolean)                                     │
│  ├─ created_at, updated_at                                   │
│  └─ [1:N] UserSubscription.billing_address_id                │
└──────────────────────────────────────────────────────────────┘
```

---

## 🔐 SECURITY & ISOLATION LAYERS

```
┌─────────────────────────────────────────────────────────────┐
│ 1. PUBLIC ENDPOINTS (No auth required)                       │
│    ├─ GET /payment/plans                                     │
│    ├─ GET /payment/summary                                   │
│    └─ POST /payment/midtrans/notification (Midtrans auth)    │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. PROTECTED ENDPOINTS (JWT required)                        │
│    ├─ POST /payment/checkout                                 │
│    ├─ GET /payment/status                                    │
│    └─ POST /payment/cancel                                   │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. MIDTRANS WEBHOOK AUTHENTICATION                           │
│    ├─ Validate signature: SHA512(OrderId+Status+Amount+Key)  │
│    ├─ Prevent spoofing attacks                               │
│    └─ Ensure payment status updates from legitimate source   │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. DATA OWNERSHIP VERIFICATION                               │
│    ├─ Subscription: owned_by user_id (from JWT)              │
│    ├─ BillingAddress: owned_by user_id (from JWT)            │
│    └─ Prevents cross-user data access                        │
└─────────────────────────────────────────────────────────────┘
```

---

## 🏗️ DEPENDENCY INJECTION CHAIN

```
cmd/rest/main.go
    └─ bootstrap.NewContainer(db, cfg)
        ├─ uowFactory := unitofwork.NewRepositoryFactory(db)
        │
        ├─ paymentService := service.NewPaymentService(uowFactory)
        │   └─ Depends on: Repository Factory
        │
        └─ PaymentController := controller.NewPaymentController(paymentService)
            ├─ Depends on: IPaymentService
            └─ Injected via Constructor
```

---

## 🎯 KEY DECISION POINTS

| Decision Point | Input | Logic | Output |
|---|---|---|---|
| **Plan Validation** | plan_id | Query plan, verify exists & active | Error \| Plan entity |
| **User Validation** | user_id | Query user, verify exists | Error \| User entity |
| **Order Calculation** | price, taxRate | Subtotal + (Subtotal * TaxRate) | Total amount |
| **Midtrans Integration** | checkout request | Send to Midtrans API | snapToken, redirectURL |
| **Payment Status** | transaction_status | Map to subscription status | active \| inactive |
| **Active Subscription** | subscriptions list | Find valid + in-period | activeSub or Free Plan |

---

## 📊 PAYMENT STATUS STATE MACHINE

```
              ┌────────────┐
              │  PENDING   │  (Subscription created, waiting for payment)
              └─────┬──────┘
                    │
        ┌───────────┴───────────┐
        │                       │
        ▼                       ▼
   ┌─────────┐           ┌──────────┐
   │  PAID   │           │  FAILED  │
   └────┬────┘           └──────────┘
        │
        ├─ Subscription becomes ACTIVE
        ├─ User gains plan access
        ├─ Features enabled
        └─ Period renewal scheduled
```

---

## 📁 FILE LOCATIONS REFERENCE

| Component | File Location |
|---|---|
| **Controller** | [internal/controller/payment_controller.go](internal/controller/payment_controller.go) |
| **Service** | [internal/service/payment_service.go](internal/service/payment_service.go) |
| **Plan Admin Service** | [internal/service/admin_service.go](internal/service/admin_service.go) |
| **Subscription Repository** | [internal/repository/contract/subscription_repository.go](internal/repository/contract/subscription_repository.go) |
| **Subscription Entity** | [internal/entity/subscription_entity.go](internal/entity/subscription_entity.go) |
| **Subscription Model** | [internal/model/subscription_model.go](internal/model/subscription_model.go) |
| **Payment DTOs** | [internal/dto/auth_payment_dto.go](internal/dto/auth_payment_dto.go) |
| **Database Schema** | [migrations/](migrations/) |
| **Admin API Docs** | [docs/ADMIN_SUBSCRIPTION_PLAN_API.md](docs/ADMIN_SUBSCRIPTION_PLAN_API.md) |

---

## ⚙️ CONFIGURATION & ENVIRONMENT

```env
# Midtrans Configuration
MIDTRANS_SERVER_KEY=<server_key_from_midtrans>
MIDTRANS_CLIENT_KEY=<client_key_from_midtrans>
MIDTRANS_IS_PRODUCTION=false  # true untuk production

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/notefiber

# Frontend
FRONTEND_URL=http://localhost:3000

# Currency & Localization
CURRENCY=USD
TAX_RATE=0.11  # Indonesia VAT
```

---

## 🚀 DEPLOYMENT CONSIDERATIONS

1. **Dependency Order:**
   - PostgreSQL must be running
   - Midtrans account configured
   - Server Key & Client Key stored safely

2. **Webhook Setup:**
   - Register webhook URL in Midtrans dashboard
   - Webhook must be accessible dari internet
   - Must return HTTP 200 OK immediately

3. **Payment Testing:**
   - Use Midtrans Sandbox untuk testing
   - Test credit card numbers provided by Midtrans
   - Verify webhook handling dengan test transactions

4. **Data Persistence:**
   - All subscription changes dalam database transaction
   - Billing address stored permanently
   - Payment history stored untuk audit trail

5. **Scalability:**
   - Webhook handling should be idempotent (same webhook twice = safe)
   - Payment status updates are atomic operations
   - Consider queue untuk high-volume webhooks (future)

---

## 💡 IMPORTANT NOTES

1. **Idempotency**: Webhook handler designed untuk handle duplicate webhooks safely
2. **Signature Verification**: ALWAYS validate Midtrans webhook signature
3. **Timezone**: All timestamps stored dalam UTC
4. **Period Calculation**: Monthly = +1 month, Yearly = +1 year (not 365 days)
5. **Free Plan Defaults**: When no active subscription, user gets: 3 notebooks, 10 notes, no AI

---

**Generated:** 28 December 2025  
**Framework:** Go + Fiber + Midtrans SDK  
**Architecture Pattern:** Layered Architecture + Repository Pattern + Unit of Work
