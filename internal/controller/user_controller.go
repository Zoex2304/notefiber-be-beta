// FILE: internal/controller/user_controller.go
package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type IUserController interface {
	RegisterRoutes(r fiber.Router)
	GetProfile(ctx *fiber.Ctx) error
	UpdateProfile(ctx *fiber.Ctx) error
	DeleteAccount(ctx *fiber.Ctx) error
	UploadAvatar(ctx *fiber.Ctx) error
	RequestRefund(ctx *fiber.Ctx) error
	GetRefunds(ctx *fiber.Ctx) error
	GetRefundDetail(ctx *fiber.Ctx) error

	// Billing
	GetBillingInfo(ctx *fiber.Ctx) error
	UpdateBillingInfo(ctx *fiber.Ctx) error

	// Cancellation
	RequestCancellation(ctx *fiber.Ctx) error
	GetCancellations(ctx *fiber.Ctx) error
	GetCancellationDetail(ctx *fiber.Ctx) error
}

type userController struct {
	service service.IUserService
}

func NewUserController(service service.IUserService) IUserController {
	return &userController{service: service}
}

func (c *userController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/user")
	h.Use(serverutils.JwtMiddleware) // Ensure this middleware is available or reuse the one from payment controller logic
	h.Get("/profile", c.GetProfile)
	h.Put("/profile", c.UpdateProfile)
	h.Delete("/account", c.DeleteAccount)
	h.Post("/avatar", c.UploadAvatar)
	h.Post("/refund/request", c.RequestRefund)
	h.Get("/refunds", c.GetRefunds)
	h.Get("/refunds/:id", c.GetRefundDetail)

	// Billing
	h.Get("/billing", c.GetBillingInfo)
	h.Put("/billing", c.UpdateBillingInfo)

	// Cancellation
	h.Post("/cancellation", c.RequestCancellation)
	h.Get("/cancellations", c.GetCancellations)
	h.Get("/cancellations/:id", c.GetCancellationDetail)
}

func (c *userController) GetProfile(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	res, err := c.service.GetProfile(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User profile", res))
}

func (c *userController) UpdateProfile(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	var req dto.UpdateProfileRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	err := c.service.UpdateProfile(ctx.Context(), userId, &req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Profile updated", nil))
}

func (c *userController) DeleteAccount(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	err := c.service.DeleteAccount(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Account deleted", nil))
}

func (c *userController) UploadAvatar(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	// Get file from request
	file, err := ctx.FormFile("avatar")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Image file is required"))
	}

	url, err := c.service.UploadAvatar(ctx.Context(), userId, file)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Avatar uploaded successfully", map[string]string{
		"avatar_url": url,
	}))
}

// RequestRefund handles user refund request
func (c *userController) RequestRefund(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	var req dto.UserRefundRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	if req.SubscriptionId == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "subscription_id is required"))
	}
	if req.Reason == "" || len(req.Reason) < 10 {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "reason must be at least 10 characters"))
	}

	res, err := c.service.RequestRefund(ctx.Context(), userId, req)
	if err != nil {
		// Handle specific errors
		errMsg := err.Error()
		if errMsg == "subscription not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, errMsg))
		}
		if errMsg == "refund already requested for this subscription" ||
			errMsg == "subscription is not active" ||
			errMsg == "subscription is not eligible for refund" {
			return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, errMsg))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Refund request submitted", res))
}

// GetRefunds returns all refund requests for the current user
func (c *userController) GetRefunds(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	refunds, err := c.service.GetRefunds(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("User refunds", refunds))
}

// GetRefundDetail returns details for a specific refund
func (c *userController) GetRefundDetail(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	idParam := ctx.Params("id")
	refundId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid refund ID"))
	}

	res, err := c.service.GetRefundDetail(ctx.Context(), userId, refundId)
	if err != nil {
		if err.Error() == "refund not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, "Refund not found"))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Refund details", res))
}

// --- Billing Management Endpoints ---

// GetBillingInfo returns the user's default billing address for Settings page
func (c *userController) GetBillingInfo(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	billing, err := c.service.GetBillingInfo(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	if billing == nil {
		return ctx.JSON(serverutils.SuccessResponse[any]("No billing info found", nil))
	}

	return ctx.JSON(serverutils.SuccessResponse("User billing info", billing))
}

// UpdateBillingInfo updates the user's billing information
func (c *userController) UpdateBillingInfo(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	var req dto.UserBillingUpdateRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	if err := c.service.UpdateBillingInfo(ctx.Context(), userId, req); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse[any]("Billing info updated", nil))
}

// --- Cancellation Management Endpoints ---

// RequestCancellation creates a cancellation request for user's subscription
func (c *userController) RequestCancellation(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	var req dto.UserCancellationRequest
	if err := ctx.BodyParser(&req); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid request body"))
	}

	if req.SubscriptionId == uuid.Nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "subscription_id is required"))
	}
	if req.Reason == "" || len(req.Reason) < 10 {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "reason must be at least 10 characters"))
	}

	res, err := c.service.RequestCancellation(ctx.Context(), userId, req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "subscription not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, errMsg))
		}
		if errMsg == "cancellation already requested for this subscription" ||
			errMsg == "subscription is not active" {
			return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, errMsg))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Cancellation request submitted", res))
}

// GetCancellations returns all cancellation requests for the current user
func (c *userController) GetCancellations(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	cancellations, err := c.service.GetCancellations(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("User cancellations", cancellations))
}

// GetCancellationDetail returns details for a specific cancellation
func (c *userController) GetCancellationDetail(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	idParam := ctx.Params("id")
	cancellationId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid cancellation ID"))
	}

	res, err := c.service.GetCancellationDetail(ctx.Context(), userId, cancellationId)
	if err != nil {
		if err.Error() == "cancellation not found" {
			return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, "Cancellation not found"))
		}
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Cancellation details", res))
}
