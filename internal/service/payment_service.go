// FILE: internal/service/payment_service.go
package service

import (
	"context"
	"crypto/sha512"
	"errors"
	"fmt"
	"os"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"ai-notetaking-be/pkg/events"
	pktNats "ai-notetaking-be/pkg/nats" // Renamed to avoid collision

	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

type IPaymentService interface {
	GetPlans(ctx context.Context) ([]*dto.PlanResponse, error)
	GetOrderSummary(ctx context.Context, planId uuid.UUID) (*dto.OrderSummaryResponse, error)
	CreateSubscription(ctx context.Context, userId uuid.UUID, req *dto.CheckoutRequest) (*dto.CheckoutResponse, error)
	HandleNotification(ctx context.Context, req *dto.MidtransWebhookRequest) error
	GetSubscriptionStatus(ctx context.Context, userId uuid.UUID) (*dto.SubscriptionStatusResponse, error)
	CancelSubscription(ctx context.Context, userId uuid.UUID) error
	ValidateSubscription(ctx context.Context, userId uuid.UUID) (*dto.SubscriptionValidationResponse, error)
}

type paymentService struct {
	uowFactory     unitofwork.RepositoryFactory
	eventPublisher *pktNats.Publisher
}

func NewPaymentService(uowFactory unitofwork.RepositoryFactory, eventPublisher *pktNats.Publisher) IPaymentService {
	return &paymentService{
		uowFactory:     uowFactory,
		eventPublisher: eventPublisher,
	}
}

func (s *paymentService) GetPlans(ctx context.Context) ([]*dto.PlanResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	// We might need an OrderBy spec for plans? Legacy didn't specify.
	plans, err := uow.SubscriptionRepository().FindAllPlans(ctx)
	if err != nil {
		return nil, err
	}

	var res []*dto.PlanResponse
	for _, p := range plans {
		features := []string{"Basic Note Taking"}
		if p.SemanticSearchEnabled {
			features = append(features, "Semantic Search")
		}
		if p.AiChatEnabled {
			features = append(features, "AI Chat Assistant")
		}

		res = append(res, &dto.PlanResponse{
			Id:          p.Id,
			Name:        p.Name,
			Slug:        p.Slug,
			Price:       p.Price,
			Description: p.Description,
			Features:    features,
		})
	}
	return res, nil
}

func (s *paymentService) GetOrderSummary(ctx context.Context, planId uuid.UUID) (*dto.OrderSummaryResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: planId})
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errors.New("plan not found")
	}

	subtotal := plan.Price
	taxRate := plan.TaxRate
	tax := subtotal * taxRate
	total := subtotal + tax

	billingPeriod := "month"
	if plan.BillingPeriod == entity.BillingPeriodYearly {
		billingPeriod = "year"
	}

	return &dto.OrderSummaryResponse{
		PlanName:      plan.Name,
		BillingPeriod: billingPeriod,
		PricePerUnit:  fmt.Sprintf("$%.2f/%s", plan.Price, billingPeriod),
		Subtotal:      subtotal,
		Tax:           tax,
		Total:         total,
		Currency:      "USD",
	}, nil
}

func (s *paymentService) CreateSubscription(ctx context.Context, userId uuid.UUID, req *dto.CheckoutRequest) (*dto.CheckoutResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: req.PlanId})
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errors.New("plan not found")
	}

	// Use generic FindOne
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	billingId := uuid.New()
	billingAddr := &entity.BillingAddress{
		Id:           billingId,
		UserId:       userId,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		Phone:        req.Phone,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      req.Country,
		IsDefault:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	subId := uuid.New()
	sub := &entity.UserSubscription{
		Id:                 subId,
		UserId:             userId,
		PlanId:             plan.Id,
		BillingAddressId:   &billingId,
		Status:             entity.SubscriptionStatusInactive,
		PaymentStatus:      entity.PaymentStatusPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
	}

	if plan.BillingPeriod == entity.BillingPeriodYearly {
		sub.CurrentPeriodEnd = time.Now().AddDate(1, 0, 0)
	}

	// Transaction
	if err := uow.Begin(ctx); err != nil {
		return nil, err
	}
	defer uow.Rollback()

	if err := uow.BillingRepository().Create(ctx, billingAddr); err != nil {
		return nil, fmt.Errorf("failed to save billing address: %v", err)
	}

	if err := uow.SubscriptionRepository().CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	if err := uow.Commit(); err != nil {
		return nil, err
	}

	// -- Midtrans Logic (External Service calls usually outside DB tx, safe here after commit) --
	var sClient snap.Client
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	env := midtrans.Sandbox
	if os.Getenv("MIDTRANS_IS_PRODUCTION") == "true" {
		env = midtrans.Production
	}
	sClient.New(serverKey, env)

	frontendURL := os.Getenv("FRONTEND_URL")
	finishRedirectURL := fmt.Sprintf("%s/app?payment=success", frontendURL)

	taxRate := plan.TaxRate
	finalAmount := int64(plan.Price + (plan.Price * taxRate))

	midtransPostalCode := req.PostalCode
	if len(midtransPostalCode) > 5 {
		midtransPostalCode = midtransPostalCode[:5]
	}

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  subId.String(),
			GrossAmt: finalAmount,
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		Callbacks: &snap.Callbacks{
			Finish: finishRedirectURL,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: req.FirstName,
			LName: req.LastName,
			Email: req.Email,
			Phone: req.Phone,
			BillAddr: &midtrans.CustomerAddress{
				FName:       req.FirstName,
				LName:       req.LastName,
				Phone:       req.Phone,
				Address:     req.AddressLine1,
				City:        req.City,
				Postcode:    midtransPostalCode,
				CountryCode: "IDN",
			},
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    plan.Id.String(),
				Price: int64(plan.Price),
				Qty:   1,
				Name:  plan.Name,
			},
		},
		EnabledPayments: snap.AllSnapPaymentType,
	}

	snapResp, midErr := sClient.CreateTransaction(snapReq)
	if midErr != nil {
		return nil, fmt.Errorf("midtrans error: %v", midErr.GetMessage())
	}

	// Emit SUBSCRIPTION_CREATED event
	if s.eventPublisher != nil {
		evt := events.BaseEvent{
			Type: "SUBSCRIPTION_CREATED",
			Data: map[string]interface{}{
				"plan_name":   plan.Name,
				"user_id":     userId,
				"full_name":   user.FullName,
				"avatar_url":  user.AvatarURL,
				"plan_id":     plan.Id,
				"amount":      plan.Price,
				"currency":    "USD", // Assuming USD
				"occurred_at": time.Now(),
			},
			OccurredAt: time.Now(),
		}
		if err := s.eventPublisher.Publish(ctx, evt); err != nil {
			fmt.Printf("[WARN] Failed to publish SUBSCRIPTION_CREATED event: %v\n", err)
		}
	}

	return &dto.CheckoutResponse{
		SubscriptionId:  subId,
		SnapToken:       snapResp.Token,
		SnapRedirectUrl: snapResp.RedirectURL,
	}, nil
}

func (s *paymentService) HandleNotification(ctx context.Context, req *dto.MidtransWebhookRequest) error {
	fmt.Printf("\n[WEBHOOK] ========== Processing Notification ==========\n")
	fmt.Printf("[WEBHOOK] OrderId: %s | Status: %s\n", req.OrderId, req.TransactionStatus)

	// P2: Signature Validation
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	if serverKey == "" {
		fmt.Println("[WEBHOOK ERROR] MIDTRANS_SERVER_KEY not configured")
		return fmt.Errorf("server configuration error")
	}

	// Midtrans signature = SHA512(order_id + status_code + gross_amount + server_key)
	signatureInput := req.OrderId + req.StatusCode + req.GrossAmount + serverKey
	expectedSignature := fmt.Sprintf("%x", sha512.Sum512([]byte(signatureInput)))

	if req.SignatureKey != expectedSignature {
		fmt.Printf("[WEBHOOK ERROR] Signature mismatch for OrderId=%s\n", req.OrderId)
		fmt.Printf("[WEBHOOK DEBUG] Expected: %s...\n", expectedSignature[:16])
		fmt.Printf("[WEBHOOK DEBUG] Received: %s...\n", req.SignatureKey[:16])
		return fmt.Errorf("invalid signature")
	}
	fmt.Printf("[WEBHOOK] Signature validated successfully\n")

	// Parse subscription ID from order_id
	subId, err := uuid.Parse(req.OrderId)
	if err != nil {
		fmt.Printf("[WEBHOOK ERROR] Invalid order_id format: %s\n", req.OrderId)
		return fmt.Errorf("invalid order id format")
	}

	// P1: Transaction wrapper
	uow := s.uowFactory.NewUnitOfWork(ctx)
	if err := uow.Begin(ctx); err != nil {
		fmt.Printf("[WEBHOOK ERROR] Failed to begin transaction: %v\n", err)
		return err
	}
	defer uow.Rollback()

	sub, err := uow.SubscriptionRepository().FindOneSubscription(ctx, specification.ByID{ID: subId})
	if err != nil {
		fmt.Printf("[WEBHOOK ERROR] Database error finding subscription: %v\n", err)
		return err
	}
	if sub == nil {
		fmt.Printf("[WEBHOOK ERROR] Subscription not found: %s\n", req.OrderId)
		return fmt.Errorf("subscription not found")
	}

	fmt.Printf("[WEBHOOK] Found subscription: UserId=%s, CurrentStatus=%s, PaymentStatus=%s\n",
		sub.UserId, sub.Status, sub.PaymentStatus)

	// Determine new status based on transaction status
	var newStatus entity.SubscriptionStatus
	var newPaymentStatus entity.PaymentStatus

	switch req.TransactionStatus {
	case "capture", "settlement":
		newStatus = entity.SubscriptionStatusActive
		newPaymentStatus = entity.PaymentStatusPaid
		fmt.Printf("[WEBHOOK] Payment SUCCESS - will activate subscription\n")
	case "deny", "cancel", "expire":
		newStatus = entity.SubscriptionStatusInactive
		newPaymentStatus = entity.PaymentStatusFailed
		fmt.Printf("[WEBHOOK] Payment FAILED - will deactivate subscription\n")
	case "pending":
		fmt.Printf("[WEBHOOK] Payment PENDING - no action needed\n")
		return nil
	default:
		fmt.Printf("[WEBHOOK] Unknown status '%s' - no action taken\n", req.TransactionStatus)
		return nil
	}

	// Check if update is needed
	if sub.Status == newStatus && sub.PaymentStatus == newPaymentStatus {
		fmt.Printf("[WEBHOOK] Status already up-to-date, skipping update\n")
		return nil
	}

	// P3: Log state transition
	fmt.Printf("[WEBHOOK] State transition: Status(%s→%s), PaymentStatus(%s→%s)\n",
		sub.Status, newStatus, sub.PaymentStatus, newPaymentStatus)

	// Apply changes
	sub.Status = newStatus
	sub.PaymentStatus = newPaymentStatus

	if err := uow.SubscriptionRepository().UpdateSubscription(ctx, sub); err != nil {
		fmt.Printf("[WEBHOOK ERROR] Failed to update subscription: %v\n", err)
		return err
	}

	if err := uow.Commit(); err != nil {
		fmt.Printf("[WEBHOOK ERROR] Failed to commit transaction: %v\n", err)
		return err
	}

	fmt.Printf("[WEBHOOK] ✅ Successfully updated subscription %s\n", subId)
	fmt.Printf("[WEBHOOK] ===========================================\n\n")
	return nil
}

func (s *paymentService) GetSubscriptionStatus(ctx context.Context, userId uuid.UUID) (*dto.SubscriptionStatusResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	// Assuming "ActiveByUserId" means status=active and user_id=userId
	// NOTE: Need to support finding by UserID AND Status.
	// I need a Specification for ByStatus? Or SubscriptionStatus spec.
	// Or I can just FindAll and filter? (bad).
	// Let's create a specific Specification for Subscription?
	// Or just use generic Where clause via custom spec if needed, OR add `ByStatus` to common specs?
	// `ActiveUsers` exists in `user_specifications`.
	// I'll assume we can find by UserID and inspect results, typically user has 1 active sub.
	// Let's assume finding LAST created subscription for user?
	// The helper `GetActiveByUserId` in legacy likely did: WHERE user_id = ? AND status = 'active' ORDER BY created_at DESC LIMIT 1.

	// Create Custom Spec inline or assume I can fetch all for user.
	// Let's create `specification.UserOwnedBy` (Already exists).
	// And `specification.SubscriptionStatusIs{Status}`?
	// I'll rely on `UserOwnedBy` and manual filter if spec missing, or add spec.
	// Actually better to just add `Specification.ByStatus`?
	// Let's use `specification.UserOwnedBy` and filter in code (assuming low volume) OR
	// assume I added `specification.ByStatus`.

	// I will just use `UserOwnedBy` and pick the active one.
	// Order by created_at DESC to get most recent subscriptions first
	subs, err := uow.SubscriptionRepository().FindAllSubscriptions(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return nil, err
	}

	// Find the most recent active subscription with valid period
	var activeSub *entity.UserSubscription
	for _, sub := range subs {
		// First priority: active status with valid period
		if sub.Status == entity.SubscriptionStatusActive && sub.CurrentPeriodEnd.After(time.Now()) {
			activeSub = sub
			break
		}
	}

	// If no active found, check if there's a recently paid one that should be active
	if activeSub == nil {
		for _, sub := range subs {
			// Second priority: payment succeeded but status might not be updated yet
			if sub.PaymentStatus == entity.PaymentStatusPaid && sub.CurrentPeriodEnd.After(time.Now()) {
				activeSub = sub
				break
			}
		}
	}

	if activeSub == nil {
		return &dto.SubscriptionStatusResponse{
			PlanName: "Free Plan",
			Status:   "inactive",
			IsActive: false,
			Features: dto.SubscriptionFeatures{
				AiChat:              false,
				SemanticSearch:      false,
				MaxNotebooks:        3,
				MaxNotesPerNotebook: 10,
			},
		}, nil
	}

	// Get Plan
	plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: activeSub.PlanId})
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errors.New("plan not found for active subscription")
	}

	// Check for pending refund
	hasPendingRefund := false
	refunds, err := uow.RefundRepository().FindAll(ctx,
		specification.Filter("subscription_id", activeSub.Id),
		specification.Filter("status", "pending"),
	)
	if err == nil && len(refunds) > 0 {
		hasPendingRefund = true
	}

	return &dto.SubscriptionStatusResponse{
		SubscriptionId:           activeSub.Id,
		PlanName:                 plan.Name,
		Status:                   string(activeSub.Status),
		CurrentPeriodEnd:         activeSub.CurrentPeriodEnd,
		AiChatDailyLimit:         plan.AiChatDailyLimit,
		SemanticSearchDailyLimit: plan.SemanticSearchDailyLimit,
		IsActive:                 true,
		HasPendingRefund:         hasPendingRefund,
		Features: dto.SubscriptionFeatures{
			AiChat:              plan.AiChatEnabled,
			SemanticSearch:      plan.SemanticSearchEnabled,
			MaxNotebooks:        plan.MaxNotebooks,
			MaxNotesPerNotebook: plan.MaxNotesPerNotebook,
		},
	}, nil
}

func (s *paymentService) CancelSubscription(ctx context.Context, userId uuid.UUID) error {
	// Look for active sub
	uow := s.uowFactory.NewUnitOfWork(ctx)
	subs, err := uow.SubscriptionRepository().FindAllSubscriptions(ctx, specification.UserOwnedBy{UserID: userId})
	if err != nil {
		return err
	}

	var activeSub *entity.UserSubscription
	for _, sub := range subs {
		if sub.Status == entity.SubscriptionStatusActive {
			activeSub = sub
			break
		}
	}

	if activeSub == nil {
		return errors.New("no active subscription found")
	}

	activeSub.Status = entity.SubscriptionStatusCanceled
	return uow.SubscriptionRepository().UpdateSubscription(ctx, activeSub)
}

// ValidateSubscription checks if user's subscription is valid (lazy evaluation approach)
// This endpoint is called by frontend to determine subscription status without cronjobs
func (s *paymentService) ValidateSubscription(ctx context.Context, userId uuid.UUID) (*dto.SubscriptionValidationResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	// Find user's most recent subscription
	subs, err := uow.SubscriptionRepository().FindAllSubscriptions(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return nil, err
	}

	// No subscription found - user is on free tier
	if len(subs) == 0 {
		return &dto.SubscriptionValidationResponse{
			IsValid:         false,
			Status:          "free_tier",
			RenewalRequired: false,
		}, nil
	}

	// Find the most recent active or paid subscription
	var activeSub *entity.UserSubscription
	for _, sub := range subs {
		// Priority 1: Active
		if sub.Status == entity.SubscriptionStatusActive && sub.PaymentStatus == entity.PaymentStatusPaid {
			activeSub = sub
			break
		}
		// Priority 2: Canceled but still within billing period (access retained)
		if sub.Status == entity.SubscriptionStatusCanceled && sub.CurrentPeriodEnd.After(time.Now()) {
			activeSub = sub
			break
		}
	}

	// No active subscription - check for most recent subscription status
	if activeSub == nil {
		// Check if there's a canceled subscription
		for _, sub := range subs {
			if sub.Status == entity.SubscriptionStatusCanceled {
				return &dto.SubscriptionValidationResponse{
					IsValid:         false,
					Status:          "canceled",
					RenewalRequired: true,
				}, nil
			}
		}

		return &dto.SubscriptionValidationResponse{
			IsValid:         false,
			Status:          "inactive",
			RenewalRequired: true,
		}, nil
	}

	now := time.Now()
	periodEnd := activeSub.CurrentPeriodEnd

	// Calculate days remaining
	daysRemaining := int(periodEnd.Sub(now).Hours() / 24)
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	// Get plan name
	planName := ""
	plan, _ := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: activeSub.PlanId})
	if plan != nil {
		planName = plan.Name
	}

	// Check if subscription is still valid
	if now.Before(periodEnd) {
		// Active and valid
		return &dto.SubscriptionValidationResponse{
			IsValid:          true,
			Status:           "active",
			RenewalRequired:  false,
			CurrentPeriodEnd: periodEnd,
			DaysRemaining:    daysRemaining,
			PlanName:         planName,
		}, nil
	}

	// Subscription has expired - check for grace period (7 days)
	gracePeriodDays := 7
	gracePeriodEnd := periodEnd.AddDate(0, 0, gracePeriodDays)

	if now.Before(gracePeriodEnd) {
		// In grace period
		return &dto.SubscriptionValidationResponse{
			IsValid:          false,
			Status:           "grace_period",
			RenewalRequired:  true,
			CurrentPeriodEnd: periodEnd,
			DaysRemaining:    0,
			GracePeriodEnd:   &gracePeriodEnd,
			PlanName:         planName,
		}, nil
	}

	// Fully expired - user should renew
	return &dto.SubscriptionValidationResponse{
		IsValid:          false,
		Status:           "expired",
		RenewalRequired:  true,
		CurrentPeriodEnd: periodEnd,
		DaysRemaining:    0,
		PlanName:         planName,
	}, nil
}
