package subscription

import (
	"context"
	"fmt"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"github.com/google/uuid"
)

// UpgradeResult contains upgrade operation results
type UpgradeResult struct {
	OldSubscriptionId uuid.UUID
	NewSubscriptionId uuid.UUID
	CreditApplied     float64
	AmountDue         float64
}

// RefundResult contains refund operation results
type RefundResult struct {
	RefundId       uuid.UUID
	RefundedAmount float64
}

// Manager handles subscription-related admin operations
type Manager struct {
	logger logger.ILogger
}

// NewManager creates a new subscription manager
func NewManager(logger logger.ILogger) *Manager {
	return &Manager{
		logger: logger,
	}
}

// Upgrade handles subscription upgrade with proration
func (m *Manager) Upgrade(ctx context.Context, uow unitofwork.UnitOfWork, req dto.AdminSubscriptionUpgradeRequest) (*UpgradeResult, error) {
	// 1. Validate User
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: req.UserId})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// 2. Get New Plan
	newPlan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: req.NewPlanId})
	if err != nil {
		return nil, err
	}
	if newPlan == nil {
		return nil, fmt.Errorf("target plan not found")
	}

	// 3. Find Active Subscription
	specs := []specification.Specification{
		specification.Filter("user_id", req.UserId),
		specification.Filter("status", "active"),
	}
	currentSub, err := uow.SubscriptionRepository().FindOneSubscription(ctx, specs...)
	if err != nil {
		return nil, err
	}

	// Transaction for Logic
	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	var credit float64 = 0
	var amountDue float64 = newPlan.Price

	// Proration Logic
	if currentSub != nil {
		totalDuration := currentSub.CurrentPeriodEnd.Sub(currentSub.CurrentPeriodStart)
		usedDuration := time.Since(currentSub.CurrentPeriodStart)
		remainingDuration := totalDuration - usedDuration

		if remainingDuration > 0 && totalDuration.Hours() > 0 {
			oldPlan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: currentSub.PlanId})
			if err == nil && oldPlan != nil {
				percentRemaining := remainingDuration.Hours() / totalDuration.Hours()
				credit = oldPlan.Price * percentRemaining
			}
		}

		currentSub.Status = "canceled"
		if err := uow.SubscriptionRepository().UpdateSubscription(ctx, currentSub); err != nil {
			return nil, err
		}
	}

	// Apply Credit
	if credit > amountDue {
		credit = amountDue
	}
	amountDue -= credit

	// Create New Subscription
	var billingAddrId *uuid.UUID
	if currentSub != nil && currentSub.BillingAddressId != nil {
		billingAddrId = currentSub.BillingAddressId
	}

	newSub := &entity.UserSubscription{
		UserId:             req.UserId,
		PlanId:             newPlan.Id,
		BillingAddressId:   billingAddrId,
		Status:             "active",
		PaymentStatus:      "paid",
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
	}
	if newPlan.BillingPeriod == "yearly" {
		newSub.CurrentPeriodEnd = time.Now().AddDate(1, 0, 0)
	}

	if err := uow.SubscriptionRepository().CreateSubscription(ctx, newSub); err != nil {
		return nil, err
	}

	m.logger.Info("ADMIN", "Upgraded User Subscription", map[string]interface{}{
		"userId":  req.UserId.String(),
		"newPlan": newPlan.Name,
		"credit":  credit,
	})

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	oldSubId := uuid.Nil
	if currentSub != nil {
		oldSubId = currentSub.Id
	}

	return &UpgradeResult{
		OldSubscriptionId: oldSubId,
		NewSubscriptionId: newSub.Id,
		CreditApplied:     credit,
		AmountDue:         amountDue,
	}, nil
}

// Refund processes admin-initiated subscription refund
func (m *Manager) Refund(ctx context.Context, uow unitofwork.UnitOfWork, req dto.AdminRefundRequest) (*RefundResult, error) {
	sub, err := uow.SubscriptionRepository().FindOneSubscription(ctx, specification.ByID{ID: req.SubscriptionId})
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}

	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	sub.Status = "canceled"
	sub.PaymentStatus = "refunded"
	sub.CurrentPeriodEnd = time.Now()

	if err := uow.SubscriptionRepository().UpdateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	// Calculate refund amount
	refundAmt := 0.0
	if req.Amount != nil {
		refundAmt = *req.Amount
	} else {
		// If no amount specified, use subscription price
		plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: sub.PlanId})
		if err == nil && plan != nil {
			refundAmt = plan.Price
		}
	}

	// Create refund record for audit trail
	refundId := uuid.New()
	refund := &entity.Refund{
		ID:             refundId,
		SubscriptionID: sub.Id,
		UserID:         sub.UserId,
		Amount:         refundAmt,
		Reason:         req.Reason,
		Status:         "processed",
		CreatedAt:      time.Now(),
	}

	if err := uow.RefundRepository().Create(ctx, refund); err != nil {
		return nil, err
	}

	m.logger.Info("ADMIN", "Refunded Subscription", map[string]interface{}{
		"subscriptionId": sub.Id.String(),
		"refundId":       refundId.String(),
		"amount":         refundAmt,
		"reason":         req.Reason,
	})

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	return &RefundResult{
		RefundId:       refundId,
		RefundedAmount: refundAmt,
	}, nil
}
