// FILE: internal/controller/admin_controller.go
package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type IAdminController interface {
	RegisterRoutes(r fiber.Router)
	Login(ctx *fiber.Ctx) error // New
	GetDashboardStats(ctx *fiber.Ctx) error
	GetUserGrowth(ctx *fiber.Ctx) error
	GetAllUsers(ctx *fiber.Ctx) error
	CreateUser(ctx *fiber.Ctx) error      // New
	BulkCreateUsers(ctx *fiber.Ctx) error // New
	GetUserDetail(ctx *fiber.Ctx) error
	UpdateUserStatus(ctx *fiber.Ctx) error
	UpdateUser(ctx *fiber.Ctx) error // New
	DeleteUser(ctx *fiber.Ctx) error // New
	PurgeUsers(ctx *fiber.Ctx) error // New, Deep Purge
	GetTransactions(ctx *fiber.Ctx) error
	GetLogs(ctx *fiber.Ctx) error
	GetLogDetail(ctx *fiber.Ctx) error
	UpgradeSubscription(ctx *fiber.Ctx) error
	RefundSubscription(ctx *fiber.Ctx) error

	CreatePlan(ctx *fiber.Ctx) error
	UpdatePlan(ctx *fiber.Ctx) error
	DeletePlan(ctx *fiber.Ctx) error
	GetAllPlans(ctx *fiber.Ctx) error

	// Plan Feature Management
	GetPlanFeatures(ctx *fiber.Ctx) error
	CreatePlanFeature(ctx *fiber.Ctx) error
	DeletePlanFeature(ctx *fiber.Ctx) error

	// Feature Catalog Management (master catalog)
	GetAllFeatures(ctx *fiber.Ctx) error
	CreateFeature(ctx *fiber.Ctx) error
	UpdateFeature(ctx *fiber.Ctx) error
	DeleteFeature(ctx *fiber.Ctx) error

	// Refund Management
	GetRefunds(ctx *fiber.Ctx) error
	ApproveRefund(ctx *fiber.Ctx) error
	RejectRefund(ctx *fiber.Ctx) error

	// Token Usage Tracking
	GetTokenUsage(ctx *fiber.Ctx) error

	// AI Configuration Management
	GetAllAiConfigurations(ctx *fiber.Ctx) error
	UpdateAiConfiguration(ctx *fiber.Ctx) error
	GetAllNuances(ctx *fiber.Ctx) error
	CreateNuance(ctx *fiber.Ctx) error
	UpdateNuance(ctx *fiber.Ctx) error
	DeleteNuance(ctx *fiber.Ctx) error

	// Billing Management
	GetUserBillingAddresses(ctx *fiber.Ctx) error
	CreateBillingAddress(ctx *fiber.Ctx) error
	UpdateBillingAddress(ctx *fiber.Ctx) error
	DeleteBillingAddress(ctx *fiber.Ctx) error

	// Cancellation Management
	GetCancellations(ctx *fiber.Ctx) error
	ProcessCancellation(ctx *fiber.Ctx) error
}

type adminController struct {
	service     service.IAdminService
	authService service.IAuthService // New Dependency
}

func NewAdminController(service service.IAdminService, authService service.IAuthService) IAdminController {
	return &adminController{
		service:     service,
		authService: authService,
	}
}

// Middleware to check for Admin Role
// This logic assumes JWT claims have "role": "admin"
func (c *adminController) adminMiddleware(ctx *fiber.Ctx) error {
	authHeader := ctx.Get("Authorization")

	// Check if Authorization header exists and has Bearer prefix
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return ctx.Status(fiber.StatusUnauthorized).JSON(serverutils.ErrorResponse(401, "Missing or invalid authorization header"))
	}
	tokenStr := authHeader[7:]

	// Get JWT secret
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret"
	}

	// Parse with Claims
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || token == nil || !token.Valid {
		return ctx.Status(fiber.StatusUnauthorized).JSON(serverutils.ErrorResponse(401, "Invalid or expired token"))
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(serverutils.ErrorResponse(401, "Invalid token claims"))
	}

	// Check admin role
	role, ok := claims["role"].(string)
	if !ok {
		return ctx.Status(fiber.StatusForbidden).JSON(serverutils.ErrorResponse(403, "Access denied: Role missing"))
	}
	if role != "admin" {
		return ctx.Status(fiber.StatusForbidden).JSON(serverutils.ErrorResponse(403, "Access denied: Admins only"))
	}

	// Store user_id in context for handlers
	if userId, exists := claims["user_id"]; exists {
		ctx.Locals("user_id", userId)
	}

	return ctx.Next()
}

func (c *adminController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/admin")

	// Public Admin Route (Login)
	h.Post("/login", c.Login)

	// Protected Routes
	h.Use(c.adminMiddleware) // Enforce Admin Middleware for all routes below

	// Dashboard
	h.Get("/dashboard", c.GetDashboardStats)
	h.Get("/growth", c.GetUserGrowth)

	// Users
	h.Get("/users", c.GetAllUsers)
	h.Post("/users", c.CreateUser)           // New
	h.Post("/users/bulk", c.BulkCreateUsers) // New
	h.Get("/users/:id", c.GetUserDetail)
	h.Put("/users/:id/status", c.UpdateUserStatus)
	h.Put("/users/:id", c.UpdateUser)    // New
	h.Delete("/users/:id", c.DeleteUser) // New
	h.Post("/users/purge", c.PurgeUsers) // New, Deep Purge

	// Transactions
	h.Get("/transactions", c.GetTransactions)

	// Logs
	h.Get("/logs", c.GetLogs)
	h.Get("/logs/:id", c.GetLogDetail)

	// Subscription Actions
	h.Post("/subscriptions/upgrade", c.UpgradeSubscription)
	h.Post("/subscriptions/refund", c.RefundSubscription)

	// Plan Management
	h.Post("/plans", c.CreatePlan)
	h.Get("/plans", c.GetAllPlans)
	h.Put("/plans/:id", c.UpdatePlan)
	h.Delete("/plans/:id", c.DeletePlan)

	// Plan Feature Management
	h.Get("/plans/:id/features", c.GetPlanFeatures)
	h.Post("/plans/:id/features", c.CreatePlanFeature)
	h.Delete("/plans/:id/features/:featureId", c.DeletePlanFeature)

	// Feature Catalog Management (master catalog)
	h.Get("/features", c.GetAllFeatures)
	h.Post("/features", c.CreateFeature)
	h.Put("/features/:id", c.UpdateFeature)
	h.Delete("/features/:id", c.DeleteFeature)

	// Refund Management (User-requested refunds)
	h.Get("/refunds", c.GetRefunds)
	h.Post("/refunds/:id/approve", c.ApproveRefund)
	h.Post("/refunds/:id/reject", c.RejectRefund)

	// Token Usage Tracking
	h.Get("/token-usage", c.GetTokenUsage)
	h.Put("/token-usage/:userId", c.UpdateAiLimit)
	h.Delete("/token-usage/:userId", c.ResetAiLimit)
	h.Post("/token-usage/bulk", c.BulkUpdateAiLimit)
	h.Delete("/token-usage/bulk", c.BulkResetAiLimit)

	// AI Configuration Management
	h.Get("/ai/configurations", c.GetAllAiConfigurations)
	h.Put("/ai/configurations/:key", c.UpdateAiConfiguration)
	h.Get("/ai/nuances", c.GetAllNuances)
	h.Post("/ai/nuances", c.CreateNuance)
	h.Put("/ai/nuances/:id", c.UpdateNuance)
	h.Delete("/ai/nuances/:id", c.DeleteNuance)

	// Billing Management
	h.Get("/users/:id/billing", c.GetUserBillingAddresses)
	h.Post("/users/:id/billing", c.CreateBillingAddress)
	h.Put("/billing/:id", c.UpdateBillingAddress)
	h.Delete("/billing/:id", c.DeleteBillingAddress)

	// Cancellation Management
	h.Get("/cancellations", c.GetCancellations)
	h.Post("/cancellations/:id/process", c.ProcessCancellation)
}

// Login Handler
func (c *adminController) Login(ctx *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	ipAddress := ctx.IP()
	userAgent := ctx.Get("User-Agent")

	res, err := c.authService.LoginAdmin(ctx.Context(), &req, ipAddress, userAgent)
	if err != nil {
		// Differentiate errors if needed, generic 401 for security
		return ctx.Status(fiber.StatusUnauthorized).JSON(serverutils.ErrorResponse(401, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Admin login successful", res))
}

func (c *adminController) GetDashboardStats(ctx *fiber.Ctx) error {
	stats, err := c.service.GetDashboardStats(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Dashboard stats", stats))
}

func (c *adminController) GetUserGrowth(ctx *fiber.Ctx) error {
	stats, err := c.service.GetUserGrowth(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User growth stats", stats))
}

func (c *adminController) GetAllUsers(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	search := ctx.Query("q", "")

	users, err := c.service.GetAllUsers(ctx.Context(), page, limit, search)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User list", users))
}

func (c *adminController) CreateUser(ctx *fiber.Ctx) error {
	var req dto.AdminCreateUserRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	result, err := c.service.CreateUser(ctx.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return ctx.Status(fiber.StatusConflict).JSON(serverutils.ErrorResponse(409, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.Status(fiber.StatusCreated).JSON(serverutils.SuccessResponse("User created", result))
}

func (c *adminController) BulkCreateUsers(ctx *fiber.Ctx) error {
	// Parse Multipart Form
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Missing file"))
	}

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, "Failed to open file"))
	}
	defer file.Close()

	// Read content
	content := make([]byte, fileHeader.Size)
	_, err = file.Read(content)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, "Failed to read file"))
	}

	result, err := c.service.BulkCreateUsers(ctx.Context(), content)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.Status(fiber.StatusCreated).JSON(serverutils.SuccessResponse("Bulk creation completed", result))
}

func (c *adminController) GetUserDetail(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	userId, _ := uuid.Parse(idParam)

	user, err := c.service.GetUserDetail(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, "User not found"))
	}
	return ctx.JSON(serverutils.SuccessResponse("User detail", user))
}

func (c *adminController) UpdateUserStatus(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	userId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid User ID"))
	}

	var req dto.UpdateUserStatusRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	err = c.service.UpdateUserStatus(ctx.Context(), userId, req.Status)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("User status updated", nil))
}

func (c *adminController) UpdateUser(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	userId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid User ID"))
	}

	var req dto.AdminUpdateUserRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	res, err := c.service.UpdateUser(ctx.Context(), userId, req)
	if err != nil {
		// Could differentiate not found vs error
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User updated", res))
}

func (c *adminController) DeleteUser(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	userId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid User ID"))
	}

	if err := c.service.DeleteUser(ctx.Context(), userId); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("User deleted", nil))
}

func (c *adminController) PurgeUsers(ctx *fiber.Ctx) error {
	var req dto.AdminPurgeUsersRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	result, err := c.service.PurgeUsers(ctx.Context(), req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	// If partial failures exist, we still return 200 OK but with the failure details
	return ctx.JSON(serverutils.SuccessResponse("Users purged", result))
}

func (c *adminController) GetTransactions(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	status := ctx.Query("status", "")

	txs, err := c.service.GetTransactions(ctx.Context(), page, limit, status)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Transactions", txs))
}

func (c *adminController) GetLogs(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	level := ctx.Query("level", "")

	logs, err := c.service.GetSystemLogs(ctx.Context(), page, limit, level)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("System logs", logs))
}

func (c *adminController) GetLogDetail(ctx *fiber.Ctx) error {
	logId := ctx.Params("id") // Log ID is a string (MD5 hash), not UUID

	l, err := c.service.GetLogDetail(ctx.Context(), logId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, "Log not found"))
	}
	return ctx.JSON(serverutils.SuccessResponse("Log detail", l))
}

func (c *adminController) UpgradeSubscription(ctx *fiber.Ctx) error {
	var req dto.AdminSubscriptionUpgradeRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	// Validate req? (Fiber validater or Manual)
	// Simple ID check
	if req.UserId == uuid.Nil || req.NewPlanId == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "user_id and new_plan_id are required"))
	}

	resp, err := c.service.UpgradeSubscription(ctx.Context(), req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Subscription upgraded", resp))
}

func (c *adminController) RefundSubscription(ctx *fiber.Ctx) error {
	// Params? or Body?
	// DTO AdminRefundRequest has SubscriptionId.
	// Let's assume passed in Body, or URL param ID + Body.
	// DTO assumes SubscriptionId is in struct.
	// For API RESTfulness: POST /subscriptions/:id/refund is better.
	// But let's stick to Body for now or parse Param.
	idParam := ctx.Params("id")

	var req dto.AdminRefundRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	if idParam != "" {
		req.SubscriptionId = uuid.MustParse(idParam)
	}

	if req.SubscriptionId == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "subscription_id is required"))
	}

	resp, err := c.service.RefundSubscription(ctx.Context(), req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Subscription refunded", resp))
}

// --- Plan Management Endpoints ---

func (c *adminController) CreatePlan(ctx *fiber.Ctx) error {
	var req dto.AdminCreatePlanRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	result, err := c.service.CreatePlan(ctx.Context(), req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Plan created", result))
}

func (c *adminController) UpdatePlan(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	planId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid Plan ID"))
	}

	var req dto.AdminUpdatePlanRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	result, err := c.service.UpdatePlan(ctx.Context(), planId, req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Plan updated", result))
}

func (c *adminController) DeletePlan(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	planId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid Plan ID"))
	}

	if err := c.service.DeletePlan(ctx.Context(), planId); err != nil {
		if strings.Contains(err.Error(), "cannot delete plan") {
			return ctx.Status(fiber.StatusConflict).JSON(serverutils.ErrorResponse(409, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Plan deleted", nil))
}

func (c *adminController) GetAllPlans(ctx *fiber.Ctx) error {
	plans, err := c.service.GetAllPlans(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Subscription plans", plans))
}

// --- Refund Management Endpoints ---

// GetRefunds returns a paginated list of refund requests
func (c *adminController) GetRefunds(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	status := ctx.Query("status", "") // pending, approved, rejected, or empty for all

	refunds, err := c.service.GetRefunds(ctx.Context(), page, limit, status)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Refund requests", refunds))
}

// ApproveRefund approves a pending refund request
func (c *adminController) ApproveRefund(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	refundId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid Refund ID"))
	}

	var req dto.AdminApproveRefundRequest
	// Body is optional, parse if present
	_ = ctx.BodyParser(&req)

	resp, err := c.service.ApproveRefund(ctx.Context(), refundId, req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "refund request not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, errMsg))
		}
		if errMsg == "refund already processed" {
			return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, errMsg))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Refund approved", resp))
}

// RejectRefund rejects a pending refund request
func (c *adminController) RejectRefund(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	refundId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid Refund ID"))
	}

	var req dto.AdminRejectRefundRequest
	// Body is optional
	_ = ctx.BodyParser(&req)

	resp, err := c.service.RejectRefund(ctx.Context(), refundId, req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "refund request not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, errMsg))
		}
		if errMsg == "refund already processed" {
			return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, errMsg))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Refund rejected", resp))
}

// --- Token Usage Tracking Endpoints ---

// GetTokenUsage returns a paginated list of users with their AI token usage
func (c *adminController) GetTokenUsage(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "20"))

	usage, err := c.service.GetTokenUsage(ctx.Context(), page, limit)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Token usage", usage))
}

// --- Plan Feature Management Endpoints ---

// GetPlanFeatures returns all features for a plan
func (c *adminController) GetPlanFeatures(ctx *fiber.Ctx) error {
	planId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid plan ID"))
	}

	features, err := c.service.GetPlanFeatures(ctx.Context(), planId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Plan features", features))
}

// CreatePlanFeature adds a new feature to a plan
func (c *adminController) CreatePlanFeature(ctx *fiber.Ctx) error {
	planId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid plan ID"))
	}

	var req dto.CreatePlanFeatureRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	feature, err := c.service.CreatePlanFeature(ctx.Context(), planId, req)
	if err != nil {
		if err.Error() == "plan not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.Status(fiber.StatusCreated).JSON(serverutils.SuccessResponse("Feature created", feature))
}

// UpdatePlanFeature removed

// DeletePlanFeature removes a feature from a plan
func (c *adminController) DeletePlanFeature(ctx *fiber.Ctx) error {
	planId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid plan ID"))
	}

	featureId, err := uuid.Parse(ctx.Params("featureId"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid feature ID"))
	}

	if err := c.service.DeletePlanFeature(ctx.Context(), planId, featureId); err != nil {
		if err.Error() == "feature not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Feature deleted", nil))
}

// --- Feature Catalog Handlers (Master Catalog) ---

func (c *adminController) GetAllFeatures(ctx *fiber.Ctx) error {
	features, err := c.service.GetAllFeatures(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Feature catalog", features))
}

func (c *adminController) CreateFeature(ctx *fiber.Ctx) error {
	var req dto.CreateFeatureRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	feature, err := c.service.CreateFeature(ctx.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return ctx.Status(fiber.StatusConflict).JSON(serverutils.ErrorResponse(409, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.Status(fiber.StatusCreated).JSON(serverutils.SuccessResponse("Feature created", feature))
}

func (c *adminController) UpdateFeature(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	featureId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid feature ID"))
	}

	var req dto.UpdateFeatureRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	feature, err := c.service.UpdateFeature(ctx.Context(), featureId, req)
	if err != nil {
		if err.Error() == "feature not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Feature updated", feature))
}

func (c *adminController) DeleteFeature(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	featureId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid feature ID"))
	}

	if err := c.service.DeleteFeature(ctx.Context(), featureId); err != nil {
		if err.Error() == "feature not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Feature deleted", nil))
}

// UpdateAiLimit updates a user's AI daily limit override
func (c *adminController) UpdateAiLimit(ctx *fiber.Ctx) error {
	userIdParam := ctx.Params("userId")
	userId, err := uuid.Parse(userIdParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid user ID"))
	}

	var req dto.UpdateAiLimitRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	resp, err := c.service.UpdateAiLimit(ctx.Context(), userId, req)
	if err != nil {
		if err.Error() == "user not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("AI limit updated", resp))
}

// ResetAiLimit resets a user's AI limit to their plan default
func (c *adminController) ResetAiLimit(ctx *fiber.Ctx) error {
	userIdParam := ctx.Params("userId")
	userId, err := uuid.Parse(userIdParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid user ID"))
	}

	limitResp, err := c.service.ResetAiLimit(ctx.Context(), userId)
	if err != nil {
		if err.Error() == "user not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("AI limit reset to plan default", limitResp))
}

// BulkUpdateAiLimit updates AI limits for multiple users
func (c *adminController) BulkUpdateAiLimit(ctx *fiber.Ctx) error {
	var req dto.BulkUpdateAiLimitRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	if len(req.UserIds) == 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "user_ids is required"))
	}

	resp, err := c.service.BulkUpdateAiLimit(ctx.Context(), req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Bulk AI limit update completed", resp))
}

// BulkResetAiLimit resets AI limits for multiple users
func (c *adminController) BulkResetAiLimit(ctx *fiber.Ctx) error {
	var req dto.BulkResetAiLimitRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	if len(req.UserIds) == 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "user_ids is required"))
	}

	resp, err := c.service.BulkResetAiLimit(ctx.Context(), req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Bulk AI limit reset completed", resp))
}

// --- AI Configuration Management Endpoints ---

// GetAllAiConfigurations returns all AI configuration settings
func (c *adminController) GetAllAiConfigurations(ctx *fiber.Ctx) error {
	configs, err := c.service.GetAllAiConfigurations(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("AI configurations", configs))
}

// UpdateAiConfiguration updates an AI configuration value
func (c *adminController) UpdateAiConfiguration(ctx *fiber.Ctx) error {
	key := ctx.Params("key")
	if key == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Configuration key is required"))
	}

	var req dto.UpdateAiConfigurationRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	config, err := c.service.UpdateAiConfiguration(ctx.Context(), key, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Configuration updated", config))
}

// GetAllNuances returns all AI nuances
func (c *adminController) GetAllNuances(ctx *fiber.Ctx) error {
	nuances, err := c.service.GetAllNuances(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("AI nuances", nuances))
}

// CreateNuance creates a new AI nuance
func (c *adminController) CreateNuance(ctx *fiber.Ctx) error {
	var req dto.CreateAiNuanceRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	nuance, err := c.service.CreateNuance(ctx.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return ctx.Status(fiber.StatusConflict).JSON(serverutils.ErrorResponse(409, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.Status(fiber.StatusCreated).JSON(serverutils.SuccessResponse("Nuance created", nuance))
}

// UpdateNuance updates an existing AI nuance
func (c *adminController) UpdateNuance(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	nuanceId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid nuance ID"))
	}

	var req dto.UpdateAiNuanceRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	nuance, err := c.service.UpdateNuance(ctx.Context(), nuanceId, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Nuance updated", nuance))
}

// DeleteNuance removes an AI nuance
func (c *adminController) DeleteNuance(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	nuanceId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid nuance ID"))
	}

	if err := c.service.DeleteNuance(ctx.Context(), nuanceId); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Nuance deleted", nil))
}

// --- Billing Management Endpoints ---

// GetUserBillingAddresses returns all billing addresses for a user
func (c *adminController) GetUserBillingAddresses(ctx *fiber.Ctx) error {
	userId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid user ID"))
	}

	billings, err := c.service.GetUserBillingAddresses(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User billing addresses", billings))
}

// CreateBillingAddress creates a new billing address for a user
func (c *adminController) CreateBillingAddress(ctx *fiber.Ctx) error {
	userId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid user ID"))
	}

	var req dto.AdminBillingCreateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	billing, err := c.service.CreateBillingAddress(ctx.Context(), userId, req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.Status(fiber.StatusCreated).JSON(serverutils.SuccessResponse("Billing address created", billing))
}

// UpdateBillingAddress updates a billing address
func (c *adminController) UpdateBillingAddress(ctx *fiber.Ctx) error {
	billingId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid billing ID"))
	}

	var req dto.AdminBillingUpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	billing, err := c.service.UpdateBillingAddress(ctx.Context(), billingId, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Billing address updated", billing))
}

// DeleteBillingAddress deletes a billing address
func (c *adminController) DeleteBillingAddress(ctx *fiber.Ctx) error {
	billingId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid billing ID"))
	}

	if err := c.service.DeleteBillingAddress(ctx.Context(), billingId); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Billing address deleted", nil))
}

// --- Cancellation Management Endpoints ---

// GetCancellations returns all cancellation requests
func (c *adminController) GetCancellations(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	status := ctx.Query("status", "") // pending, approved, rejected, or empty for all

	cancellations, err := c.service.GetCancellations(ctx.Context(), page, limit, status)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Cancellation requests", cancellations))
}

// ProcessCancellation processes a cancellation request (approve/reject)
func (c *adminController) ProcessCancellation(ctx *fiber.Ctx) error {
	cancellationId, err := uuid.Parse(ctx.Params("id"))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid cancellation ID"))
	}

	var req dto.AdminProcessCancellationRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	if req.Action != "approve" && req.Action != "reject" {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "action must be 'approve' or 'reject'"))
	}

	resp, err := c.service.ProcessCancellation(ctx.Context(), cancellationId, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, err.Error()))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Cancellation processed", resp))
}
