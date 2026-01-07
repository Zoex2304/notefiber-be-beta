// FILE: internal/service/user_service.go
package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"

	"ai-notetaking-be/pkg/events"
	pktNats "ai-notetaking-be/pkg/nats" // Renamed to avoid collision

	"github.com/google/uuid"
)

type IUserService interface {
	GetProfile(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userId uuid.UUID, req *dto.UpdateProfileRequest) error
	DeleteAccount(ctx context.Context, userId uuid.UUID) error
	UploadAvatar(ctx context.Context, userId uuid.UUID, file *multipart.FileHeader) (string, error)
	RequestRefund(ctx context.Context, userId uuid.UUID, req dto.UserRefundRequest) (*dto.UserRefundResponse, error)
	GetRefunds(ctx context.Context, userId uuid.UUID) ([]*dto.UserRefundListResponse, error)
	GetRefundDetail(ctx context.Context, userId uuid.UUID, refundId uuid.UUID) (*dto.UserRefundListResponse, error)

	// Billing
	GetBillingInfo(ctx context.Context, userId uuid.UUID) (*dto.UserBillingResponse, error)
	UpdateBillingInfo(ctx context.Context, userId uuid.UUID, req dto.UserBillingUpdateRequest) error

	// Cancellation
	RequestCancellation(ctx context.Context, userId uuid.UUID, req dto.UserCancellationRequest) (*dto.UserCancellationResponse, error)
	GetCancellations(ctx context.Context, userId uuid.UUID) ([]*dto.UserCancellationListResponse, error)
	GetCancellationDetail(ctx context.Context, userId uuid.UUID, cancellationId uuid.UUID) (*dto.UserCancellationListResponse, error)
}

type userService struct {
	uowFactory     unitofwork.RepositoryFactory
	eventPublisher *pktNats.Publisher
}

// NewUserService now accepts RepositoryFactory instead of direct repository
func NewUserService(uowFactory unitofwork.RepositoryFactory, eventPublisher *pktNats.Publisher) IUserService {
	return &userService{
		uowFactory:     uowFactory,
		eventPublisher: eventPublisher,
	}
}

func (s *userService) GetProfile(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	// Use GetByIdWithAvatar which we added to contract to maintain compatibility
	user, err := uow.UserRepository().GetByIdWithAvatar(ctx, userId)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	return &dto.UserProfileResponse{
		Id:           user.Id,
		Email:        user.Email,
		FullName:     user.FullName,
		Role:         string(user.Role),
		Status:       string(user.Status),
		AvatarURL:    avatarURL,
		AiDailyUsage: user.AiDailyUsage,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userId uuid.UUID, req *dto.UpdateProfileRequest) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	// Transaction could be useful here but for single update GORM is atomic enough.
	// For consistency, let's use Begin/Commit pattern if we were doing more.

	repo := uow.UserRepository()
	user, err := repo.FindOne(ctx, specification.ByID{ID: userId})
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	user.FullName = req.FullName
	return repo.Update(ctx, user)
}

func (s *userService) DeleteAccount(ctx context.Context, userId uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	// Emit USER_DELETED Event
	if s.eventPublisher != nil {
		evt := events.BaseEvent{
			Type: "USER_DELETED",
			Data: map[string]interface{}{
				"user_id":     userId,
				"occurred_at": time.Now(),
			},
			OccurredAt: time.Now(),
		}
		if err := s.eventPublisher.Publish(ctx, evt); err != nil {
			fmt.Printf("[WARN] Failed to publish USER_DELETED event: %v\n", err)
		}
	}

	return uow.UserRepository().Delete(ctx, userId)
}

func (s *userService) UploadAvatar(ctx context.Context, userId uuid.UUID, file *multipart.FileHeader) (string, error) {
	// 1. Validate File Size (e.g., Max 2MB)
	if file.Size > 2*1024*1024 {
		return "", fmt.Errorf("file too large (max 2MB)")
	}

	// 2. Open File
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// 3. Create Upload Directory
	uploadDir := "./uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	// 4. Generate Unique Filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%d%s", userId.String(), time.Now().Unix(), ext)
	dstPath := filepath.Join(uploadDir, filename)

	// 5. Save File
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	// 6. Generate Public URL
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	publicURL := fmt.Sprintf("%s/uploads/avatars/%s", baseURL, filename)

	// 7. Update User Profile in DB
	uow := s.uowFactory.NewUnitOfWork(ctx)
	err = uow.UserRepository().UpdateAvatar(ctx, userId, publicURL)
	if err != nil {
		return "", err
	}

	return publicURL, nil
}

// RequestRefund creates a new refund request for a user's subscription
func (s *userService) RequestRefund(ctx context.Context, userId uuid.UUID, req dto.UserRefundRequest) (*dto.UserRefundResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	// 1. Validate subscription exists and belongs to user
	sub, err := uow.SubscriptionRepository().FindOneSubscription(ctx,
		specification.ByID{ID: req.SubscriptionId},
		specification.Filter("user_id", userId),
	)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}

	// 2. Check subscription is active and paid
	if sub.Status != entity.SubscriptionStatusActive {
		return nil, fmt.Errorf("subscription is not active")
	}
	if sub.PaymentStatus != entity.PaymentStatusPaid {
		return nil, fmt.Errorf("subscription is not eligible for refund")
	}

	// 3. Check if refund already requested for this subscription (PENDING or APPROVED)
	// We allow re-requesting if the previous one was REJECTED.
	existingRefunds, err := uow.RefundRepository().FindAllWithDetails(ctx,
		specification.Filter("subscription_id", req.SubscriptionId),
	)
	if err != nil {
		return nil, err
	}

	for _, r := range existingRefunds {
		if r.Status == string(entity.RefundStatusPending) || r.Status == string(entity.RefundStatusApproved) {
			return nil, fmt.Errorf("refund already requested for this subscription")
		}
	}

	// 4. Get plan price for refund amount
	plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: sub.PlanId})
	if err != nil {
		return nil, err
	}
	refundAmount := 0.0
	if plan != nil {
		refundAmount = plan.Price
	}

	// 5. Create refund record with status pending
	refundId := uuid.New()
	refund := &entity.Refund{
		ID:             refundId,
		SubscriptionID: req.SubscriptionId,
		UserID:         userId,
		Amount:         refundAmount,
		Reason:         req.Reason,
		Status:         string(entity.RefundStatusPending),
		CreatedAt:      time.Now(),
	}

	if err := uow.RefundRepository().Create(ctx, refund); err != nil {
		return nil, err
	}

	// Emit REFUND_REQUESTED Event
	if s.eventPublisher != nil {
		evt := events.BaseEvent{
			Type: "REFUND_REQUESTED",
			Data: map[string]interface{}{
				"refund_id":       refundId,
				"subscription_id": req.SubscriptionId,
				"user_id":         userId,
				"reason":          req.Reason,
				"amount":          refundAmount,
				"entity_type":     "refund",
				"entity_id":       refundId.String(),
				"occurred_at":     time.Now(),
			},
			OccurredAt: time.Now(),
		}
		if err := s.eventPublisher.Publish(ctx, evt); err != nil {
			fmt.Printf("[WARN] Failed to publish REFUND_REQUESTED event: %v\n", err)
		}
	}

	return &dto.UserRefundResponse{
		RefundId: refundId.String(),
		Status:   string(entity.RefundStatusPending),
		Message:  "Your refund request has been submitted and is awaiting admin review.",
	}, nil
}

// GetRefunds returns all refund requests for a specific user
func (s *userService) GetRefunds(ctx context.Context, userId uuid.UUID) ([]*dto.UserRefundListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	refunds, err := uow.RefundRepository().FindAllWithDetails(ctx,
		specification.Filter("user_id", userId),
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return nil, err
	}

	var res []*dto.UserRefundListResponse
	for _, r := range refunds {
		// Get plan name from subscription
		planName := ""
		if r.Subscription.PlanId != uuid.Nil {
			plan, _ := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: r.Subscription.PlanId})
			if plan != nil {
				planName = plan.Name
			}
		}

		res = append(res, &dto.UserRefundListResponse{
			Id:             r.ID,
			SubscriptionId: r.SubscriptionID,
			PlanName:       planName,
			Amount:         r.Amount,
			Reason:         r.Reason,
			Status:         r.Status,
			CreatedAt:      r.CreatedAt,
		})
	}

	return res, nil
}

// GetRefundDetail returns a single refund request detail
func (s *userService) GetRefundDetail(ctx context.Context, userId uuid.UUID, refundId uuid.UUID) (*dto.UserRefundListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	refunds, err := uow.RefundRepository().FindAllWithDetails(ctx,
		specification.ByID{ID: refundId},
		specification.Filter("user_id", userId),
	)
	if err != nil {
		return nil, err
	}
	if len(refunds) == 0 {
		return nil, fmt.Errorf("refund not found")
	}
	r := refunds[0]

	// Get plan name from subscription
	planName := ""
	if r.Subscription.PlanId != uuid.Nil {
		plan, _ := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: r.Subscription.PlanId})
		if plan != nil {
			planName = plan.Name
		}
	}

	return &dto.UserRefundListResponse{
		Id:             r.ID,
		SubscriptionId: r.SubscriptionID,
		PlanName:       planName,
		Amount:         r.Amount,
		Reason:         r.Reason,
		Status:         r.Status,
		CreatedAt:      r.CreatedAt,
	}, nil
}

// ============================================================================
// Billing Management
// ============================================================================

// GetBillingInfo returns the user's default billing address for Settings page
func (s *userService) GetBillingInfo(ctx context.Context, userId uuid.UUID) (*dto.UserBillingResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	// Get default billing address for user
	billing, err := uow.BillingRepository().FindOne(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.Filter("is_default", true),
	)
	if err != nil {
		return nil, err
	}

	if billing == nil {
		// Return empty response if no billing info exists
		return nil, nil
	}

	return &dto.UserBillingResponse{
		Id:           billing.Id,
		FirstName:    billing.FirstName,
		LastName:     billing.LastName,
		Email:        billing.Email,
		Phone:        billing.Phone,
		AddressLine1: billing.AddressLine1,
		AddressLine2: billing.AddressLine2,
		City:         billing.City,
		State:        billing.State,
		PostalCode:   billing.PostalCode,
		Country:      billing.Country,
	}, nil
}

// UpdateBillingInfo updates the user's billing information
func (s *userService) UpdateBillingInfo(ctx context.Context, userId uuid.UUID, req dto.UserBillingUpdateRequest) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	// Find existing default billing or create new
	billing, err := uow.BillingRepository().FindOne(ctx,
		specification.UserOwnedBy{UserID: userId},
		specification.Filter("is_default", true),
	)
	if err != nil {
		return err
	}

	if billing == nil {
		// Create new billing address
		billing = &entity.BillingAddress{
			Id:        uuid.New(),
			UserId:    userId,
			IsDefault: true,
			CreatedAt: time.Now(),
		}
	}

	// Update fields
	billing.FirstName = req.FirstName
	billing.LastName = req.LastName
	billing.Email = req.Email
	billing.Phone = req.Phone
	billing.AddressLine1 = req.AddressLine1
	billing.AddressLine2 = req.AddressLine2
	billing.City = req.City
	billing.State = req.State
	billing.PostalCode = req.PostalCode
	billing.Country = req.Country
	billing.UpdatedAt = time.Now()

	if billing.CreatedAt.IsZero() {
		billing.CreatedAt = time.Now()
		return uow.BillingRepository().Create(ctx, billing)
	}
	return uow.BillingRepository().Update(ctx, billing)
}

// ============================================================================
// Cancellation Management
// ============================================================================

// RequestCancellation creates a new cancellation request for a user's subscription
func (s *userService) RequestCancellation(ctx context.Context, userId uuid.UUID, req dto.UserCancellationRequest) (*dto.UserCancellationResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	// 1. Validate subscription exists and belongs to user
	sub, err := uow.SubscriptionRepository().FindOneSubscription(ctx,
		specification.ByID{ID: req.SubscriptionId},
		specification.Filter("user_id", userId),
	)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, fmt.Errorf("subscription not found")
	}

	// 2. Check subscription is active
	if sub.Status != entity.SubscriptionStatusActive {
		return nil, fmt.Errorf("subscription is not active")
	}

	// 3. Check if cancellation already requested for this subscription
	existingCancellations, err := uow.CancellationRepository().FindAll(ctx,
		specification.Filter("subscription_id", req.SubscriptionId),
	)
	if err != nil {
		return nil, err
	}

	for _, c := range existingCancellations {
		if c.Status == string(entity.CancellationStatusPending) || c.Status == string(entity.CancellationStatusApproved) {
			return nil, fmt.Errorf("cancellation already requested for this subscription")
		}
	}

	// 4. Create cancellation record
	cancellationId := uuid.New()
	cancellation := &entity.Cancellation{
		ID:             cancellationId,
		SubscriptionID: req.SubscriptionId,
		UserID:         userId,
		Reason:         req.Reason,
		Status:         string(entity.CancellationStatusPending),
		EffectiveDate:  sub.CurrentPeriodEnd, // Cancellation takes effect at end of period
		CreatedAt:      time.Now(),
	}

	if err := uow.CancellationRepository().Create(ctx, cancellation); err != nil {
		return nil, err
	}

	// Emit CANCELLATION_REQUESTED Event
	if s.eventPublisher != nil {
		evt := events.BaseEvent{
			Type: "SUBSCRIPTION_CANCELLATION_REQUESTED",
			Data: map[string]interface{}{
				"cancellation_id": cancellationId,
				"subscription_id": req.SubscriptionId,
				"user_id":         userId,
				"reason":          req.Reason,
				"entity_type":     "cancellation",
				"entity_id":       cancellationId.String(),
				"occurred_at":     time.Now(),
			},
			OccurredAt: time.Now(),
		}
		if err := s.eventPublisher.Publish(ctx, evt); err != nil {
			fmt.Printf("[WARN] Failed to publish SUBSCRIPTION_CANCELLATION_REQUESTED event: %v\n", err)
		}
	}

	return &dto.UserCancellationResponse{
		CancellationId: cancellationId.String(),
		Status:         string(entity.CancellationStatusPending),
		Message:        "Your cancellation request has been submitted and is awaiting admin review.",
	}, nil
}

// GetCancellations returns all cancellation requests for a specific user
func (s *userService) GetCancellations(ctx context.Context, userId uuid.UUID) ([]*dto.UserCancellationListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	cancellations, err := uow.CancellationRepository().FindAllWithDetails(ctx,
		specification.Filter("user_id", userId),
		specification.OrderBy{Field: "created_at", Desc: true},
	)
	if err != nil {
		return nil, err
	}

	var res []*dto.UserCancellationListResponse
	for _, c := range cancellations {
		// Get plan name from subscription
		planName := ""
		if c.Subscription.PlanId != uuid.Nil {
			plan, _ := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: c.Subscription.PlanId})
			if plan != nil {
				planName = plan.Name
			}
		}

		res = append(res, &dto.UserCancellationListResponse{
			Id:             c.ID,
			SubscriptionId: c.SubscriptionID,
			PlanName:       planName,
			Reason:         c.Reason,
			Status:         c.Status,
			EffectiveDate:  c.EffectiveDate,
			CreatedAt:      c.CreatedAt,
		})
	}

	return res, nil
}

// GetCancellationDetail returns a single cancellation request detail
func (s *userService) GetCancellationDetail(ctx context.Context, userId uuid.UUID, cancellationId uuid.UUID) (*dto.UserCancellationListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	cancellations, err := uow.CancellationRepository().FindAllWithDetails(ctx,
		specification.ByID{ID: cancellationId},
		specification.Filter("user_id", userId),
	)
	if err != nil {
		return nil, err
	}
	if len(cancellations) == 0 {
		return nil, fmt.Errorf("cancellation not found")
	}
	// Use the first result
	c := cancellations[0]

	// Get plan name from subscription
	planName := ""
	if c.Subscription.PlanId != uuid.Nil {
		plan, _ := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: c.Subscription.PlanId})
		if plan != nil {
			planName = plan.Name
		}
	}

	return &dto.UserCancellationListResponse{
		Id:             c.ID,
		SubscriptionId: c.SubscriptionID,
		PlanName:       planName,
		Reason:         c.Reason,
		Status:         c.Status,
		EffectiveDate:  c.EffectiveDate,
		CreatedAt:      c.CreatedAt,
	}, nil
}
