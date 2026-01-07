// FILE: internal/controller/plan_controller.go
// Controller for plan-related endpoints
package controller

import (
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type PlanController interface {
	RegisterRoutes(api fiber.Router, jwtMiddleware fiber.Handler)
}

type planController struct {
	planService service.PlanService
}

func NewPlanController(planService service.PlanService) PlanController {
	return &planController{
		planService: planService,
	}
}

func (c *planController) RegisterRoutes(api fiber.Router, jwtMiddleware fiber.Handler) {
	// Public endpoints
	api.Get("/plans", c.GetAllPlans)

	// Authenticated endpoints
	user := api.Group("/user", jwtMiddleware)
	user.Get("/usage-status", c.GetUsageStatus)
}

// GetAllPlans returns all active plans with features for pricing modal
// @Summary Get all subscription plans
// @Description Returns all active plans with their features for the pricing modal
// @Tags Plans
// @Produce json
// @Success 200 {object} []dto.PlanWithFeaturesResponse
// @Router /api/plans [get]
func (c *planController) GetAllPlans(ctx *fiber.Ctx) error {
	plans, err := c.planService.GetAllActivePlansWithFeatures(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Plans retrieved", plans))
}

// GetUsageStatus returns current usage vs limits for the authenticated user
// @Summary Get user usage status
// @Description Returns current usage counts vs plan limits for notebooks, notes, AI chat, and semantic search
// @Tags User
// @Security BearerAuth
// @Produce json
// @Success 200 {object} dto.UsageStatusResponse
// @Router /api/user/usage-status [get]
func (c *planController) GetUsageStatus(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id")
	if userIdStr == nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(serverutils.ErrorResponse(401, "Unauthorized"))
	}

	userId, err := uuid.Parse(userIdStr.(string))
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid user ID"))
	}

	status, err := c.planService.GetUserUsageStatus(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Usage status retrieved", status))
}
