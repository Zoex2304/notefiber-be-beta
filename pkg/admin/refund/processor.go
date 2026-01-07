package refund

import (
	"context"
	"fmt"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	adminEvents "ai-notetaking-be/pkg/admin/events"

	"github.com/google/uuid"
)

// ApproveResult contains approval operation results
type ApproveResult struct {
	RefundId       uuid.UUID
	RefundedAmount float64
	ProcessedAt    time.Time
}

// RejectResult contains rejection operation results
type RejectResult struct {
	RefundId    uuid.UUID
	ProcessedAt time.Time
}

// Processor handles refund approval/rejection workflow
type Processor struct {
	logger    logger.ILogger
	publisher adminEvents.Publisher
}

// NewProcessor creates a new refund processor
func NewProcessor(logger logger.ILogger, publisher adminEvents.Publisher) *Processor {
	return &Processor{
		logger:    logger,
		publisher: publisher,
	}
}

// GetAll retrieves paginated refund requests with optional status filter
func (p *Processor) GetAll(ctx context.Context, uow unitofwork.UnitOfWork, page, limit int, status string) ([]*entity.Refund, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var specs []specification.Specification
	if status != "" {
		specs = append(specs, specification.Filter("status", status))
	}
	specs = append(specs, specification.Pagination{Limit: limit, Offset: offset})
	specs = append(specs, specification.OrderBy{Field: "created_at", Desc: true})

	return uow.RefundRepository().FindAllWithDetails(ctx, specs...)
}

// GetPlanInfo retrieves plan info for a refund's subscription
func (p *Processor) GetPlanInfo(ctx context.Context, uow unitofwork.UnitOfWork, planId uuid.UUID) (string, float64) {
	plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: planId})
	if err != nil || plan == nil {
		return "", 0
	}
	return plan.Name, plan.Price
}

// Approve approves a pending refund request
func (p *Processor) Approve(ctx context.Context, uow unitofwork.UnitOfWork, refundId uuid.UUID, req dto.AdminApproveRefundRequest) (*ApproveResult, error) {
	// 1. Find the refund
	refund, err := uow.RefundRepository().FindOne(ctx, specification.ByID{ID: refundId})
	if err != nil {
		return nil, err
	}
	if refund == nil {
		return nil, fmt.Errorf("refund request not found")
	}

	// 2. Check if already processed
	if refund.Status != string(entity.RefundStatusPending) {
		return nil, fmt.Errorf("refund already processed")
	}

	// 3. Start transaction
	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	// 4. Update refund status
	now := time.Now()
	refund.Status = string(entity.RefundStatusApproved)
	refund.AdminNotes = req.AdminNotes
	refund.ProcessedAt = &now

	if err := uow.RefundRepository().Update(ctx, refund); err != nil {
		return nil, err
	}

	// 5. Update subscription status
	sub, err := uow.SubscriptionRepository().FindOneSubscription(ctx, specification.ByID{ID: refund.SubscriptionID})
	if err != nil {
		return nil, err
	}
	if sub != nil {
		sub.Status = "canceled"
		sub.PaymentStatus = "refunded"
		sub.CurrentPeriodEnd = now
		if err := uow.SubscriptionRepository().UpdateSubscription(ctx, sub); err != nil {
			return nil, err
		}
	}

	// 6. Log the action
	p.logger.Info("ADMIN", "Approved Refund Request", map[string]interface{}{
		"refundId":       refundId.String(),
		"subscriptionId": refund.SubscriptionID.String(),
		"amount":         refund.Amount,
		"adminNotes":     req.AdminNotes,
	})

	// 7. Emit REFUND_APPROVED Event
	p.publisher.PublishRefundApproved(ctx, refundId, refund.SubscriptionID, refund.UserID, refund.Amount, refund.Reason)

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	return &ApproveResult{
		RefundId:       refundId,
		RefundedAmount: refund.Amount,
		ProcessedAt:    now,
	}, nil
}

// Reject rejects a pending refund request
func (p *Processor) Reject(ctx context.Context, uow unitofwork.UnitOfWork, refundId uuid.UUID, req dto.AdminRejectRefundRequest) (*RejectResult, error) {
	// 1. Find the refund
	refund, err := uow.RefundRepository().FindOne(ctx, specification.ByID{ID: refundId})
	if err != nil {
		return nil, err
	}
	if refund == nil {
		return nil, fmt.Errorf("refund request not found")
	}

	// 2. Check if already processed
	if refund.Status != string(entity.RefundStatusPending) {
		return nil, fmt.Errorf("refund already processed")
	}

	// 3. Start transaction
	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	// 4. Update refund status
	now := time.Now()
	refund.Status = string(entity.RefundStatusRejected)
	refund.AdminNotes = req.AdminNotes
	refund.ProcessedAt = &now

	if err := uow.RefundRepository().Update(ctx, refund); err != nil {
		return nil, err
	}

	// 5. Log the action
	p.logger.Info("ADMIN", "Rejected Refund Request", map[string]interface{}{
		"refundId":       refundId.String(),
		"subscriptionId": refund.SubscriptionID.String(),
		"adminNotes":     req.AdminNotes,
	})

	// 6. Emit REFUND_REJECTED Event
	p.publisher.PublishRefundRejected(ctx, refundId, refund.SubscriptionID, refund.UserID, req.AdminNotes)

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	return &RejectResult{
		RefundId:    refundId,
		ProcessedAt: now,
	}, nil
}
