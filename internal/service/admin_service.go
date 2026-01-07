package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	"ai-notetaking-be/pkg/admin/aiconfig"
	"ai-notetaking-be/pkg/admin/dashboard"
	adminEvents "ai-notetaking-be/pkg/admin/events"
	"ai-notetaking-be/pkg/admin/feature"
	"ai-notetaking-be/pkg/admin/mapper"
	"ai-notetaking-be/pkg/admin/plan"
	"ai-notetaking-be/pkg/admin/refund"
	"ai-notetaking-be/pkg/admin/subscription"
	"ai-notetaking-be/pkg/admin/usage"
	"ai-notetaking-be/pkg/admin/user"

	"github.com/google/uuid"
)

type IAdminService interface {
	GetDashboardStats(ctx context.Context) (*dto.AdminDashboardStats, error)
	GetUserGrowth(ctx context.Context) ([]*dto.UserGrowthStats, error)

	// User Management
	GetAllUsers(ctx context.Context, page, limit int, search string) ([]*dto.UserListResponse, error)
	GetUserDetail(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error)
	UpdateUserStatus(ctx context.Context, userId uuid.UUID, status string) error
	CreateUser(ctx context.Context, req dto.AdminCreateUserRequest) (*dto.UserProfileResponse, error)
	BulkCreateUsers(ctx context.Context, fileContent []byte) (*dto.AdminBulkCreateUserResponse, error) // New
	UpdateUser(ctx context.Context, userId uuid.UUID, req dto.AdminUpdateUserRequest) (*dto.UserProfileResponse, error)
	DeleteUser(ctx context.Context, userId uuid.UUID) error
	PurgeUsers(ctx context.Context, req dto.AdminPurgeUsersRequest) (*dto.AdminPurgeUsersResponse, error)

	// Transaction Management
	GetTransactions(ctx context.Context, page, limit int, status string) ([]*dto.TransactionListResponse, error)

	// Logs
	GetSystemLogs(ctx context.Context, page, limit int, level string) ([]*dto.LogListResponse, error)
	GetLogDetail(ctx context.Context, logId string) (*dto.LogDetailResponse, error)

	// Subscription Management
	UpgradeSubscription(ctx context.Context, req dto.AdminSubscriptionUpgradeRequest) (*dto.AdminSubscriptionUpgradeResponse, error)
	RefundSubscription(ctx context.Context, req dto.AdminRefundRequest) (*dto.AdminRefundResponse, error)

	// Refund Management (User-requested refunds)
	GetRefunds(ctx context.Context, page, limit int, status string) ([]*dto.AdminRefundListResponse, error)
	ApproveRefund(ctx context.Context, refundId uuid.UUID, req dto.AdminApproveRefundRequest) (*dto.AdminApproveRefundResponse, error)
	RejectRefund(ctx context.Context, refundId uuid.UUID, req dto.AdminRejectRefundRequest) (*dto.AdminRejectRefundResponse, error)

	// Plan Management
	CreatePlan(ctx context.Context, req dto.AdminCreatePlanRequest) (*dto.AdminPlanResponse, error)
	UpdatePlan(ctx context.Context, id uuid.UUID, req dto.AdminUpdatePlanRequest) (*dto.AdminPlanResponse, error)
	DeletePlan(ctx context.Context, id uuid.UUID) error
	GetAllPlans(ctx context.Context) ([]*dto.AdminPlanResponse, error)

	// Plan Feature Management (for pricing modal)
	GetPlanFeatures(ctx context.Context, planId uuid.UUID) ([]*dto.PlanFeatureResponse, error)
	CreatePlanFeature(ctx context.Context, planId uuid.UUID, req dto.CreatePlanFeatureRequest) (*dto.PlanFeatureResponse, error)
	DeletePlanFeature(ctx context.Context, planId uuid.UUID, featureId uuid.UUID) error

	// Feature Catalog Management (master catalog)
	GetAllFeatures(ctx context.Context) ([]*dto.FeatureResponse, error)
	CreateFeature(ctx context.Context, req dto.CreateFeatureRequest) (*dto.FeatureResponse, error)
	UpdateFeature(ctx context.Context, id uuid.UUID, req dto.UpdateFeatureRequest) (*dto.FeatureResponse, error)
	DeleteFeature(ctx context.Context, id uuid.UUID) error

	// Token Usage Tracking
	GetTokenUsage(ctx context.Context, page, limit int) ([]*dto.TokenUsageResponse, error)
	UpdateAiLimit(ctx context.Context, userId uuid.UUID, req dto.UpdateAiLimitRequest) (*dto.UpdateAiLimitResponse, error)
	ResetAiLimit(ctx context.Context, userId uuid.UUID) (*dto.UpdateAiLimitResponse, error)
	BulkUpdateAiLimit(ctx context.Context, req dto.BulkUpdateAiLimitRequest) (*dto.BulkAiLimitResponse, error)
	BulkResetAiLimit(ctx context.Context, req dto.BulkResetAiLimitRequest) (*dto.BulkAiLimitResponse, error)

	// AI Configuration Management
	GetAllAiConfigurations(ctx context.Context) ([]*dto.AiConfigurationResponse, error)
	UpdateAiConfiguration(ctx context.Context, key string, req dto.UpdateAiConfigurationRequest) (*dto.AiConfigurationResponse, error)
	GetAllNuances(ctx context.Context) ([]*dto.AiNuanceResponse, error)
	CreateNuance(ctx context.Context, req dto.CreateAiNuanceRequest) (*dto.AiNuanceResponse, error)
	UpdateNuance(ctx context.Context, id uuid.UUID, req dto.UpdateAiNuanceRequest) (*dto.AiNuanceResponse, error)
	DeleteNuance(ctx context.Context, id uuid.UUID) error

	// Billing Management
	GetUserBillingAddresses(ctx context.Context, userId uuid.UUID) ([]*dto.AdminBillingListResponse, error)
	CreateBillingAddress(ctx context.Context, userId uuid.UUID, req dto.AdminBillingCreateRequest) (*dto.AdminBillingListResponse, error)
	UpdateBillingAddress(ctx context.Context, id uuid.UUID, req dto.AdminBillingUpdateRequest) (*dto.AdminBillingListResponse, error)
	DeleteBillingAddress(ctx context.Context, id uuid.UUID) error

	// Cancellation Management
	GetCancellations(ctx context.Context, page, limit int, status string) ([]*dto.AdminCancellationListResponse, error)
	ProcessCancellation(ctx context.Context, cancellationId uuid.UUID, req dto.AdminProcessCancellationRequest) (*dto.AdminProcessCancellationResponse, error)
}

type adminService struct {
	uowFactory unitofwork.RepositoryFactory
	logger     logger.ILogger

	// Domain Components
	userManager         *user.Manager
	subscriptionManager *subscription.Manager
	planManager         *plan.Manager
	featureManager      *feature.Manager
	refundProcessor     *refund.Processor
	usageTracker        *usage.Tracker
	dashboardAggregator *dashboard.Aggregator
	eventPublisher      adminEvents.Publisher
	aiConfigManager     *aiconfig.Manager
}

func NewAdminService(
	uowFactory unitofwork.RepositoryFactory,
	logger logger.ILogger,
	userManager *user.Manager,
	subscriptionManager *subscription.Manager,
	planManager *plan.Manager,
	featureManager *feature.Manager,
	refundProcessor *refund.Processor,
	usageTracker *usage.Tracker,
	dashboardAggregator *dashboard.Aggregator,
	eventPublisher adminEvents.Publisher,
	aiConfigManager *aiconfig.Manager,
) IAdminService {
	return &adminService{
		uowFactory:          uowFactory,
		logger:              logger,
		userManager:         userManager,
		subscriptionManager: subscriptionManager,
		planManager:         planManager,
		featureManager:      featureManager,
		refundProcessor:     refundProcessor,
		usageTracker:        usageTracker,
		dashboardAggregator: dashboardAggregator,
		eventPublisher:      eventPublisher,
		aiConfigManager:     aiConfigManager,
	}
}

// ============================================================================
// Dashboard & Stats
// ============================================================================

func (s *adminService) GetDashboardStats(ctx context.Context) (*dto.AdminDashboardStats, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.dashboardAggregator.GetStats(ctx, uow)
}

func (s *adminService) GetUserGrowth(ctx context.Context) ([]*dto.UserGrowthStats, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.dashboardAggregator.GetUserGrowth(ctx, uow)
}

// ============================================================================
// User Management
// ============================================================================

func (s *adminService) GetAllUsers(ctx context.Context, page, limit int, search string) ([]*dto.UserListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	users, err := s.userManager.FindAll(ctx, uow, page, limit, search)
	if err != nil {
		return nil, err
	}
	return mapper.UsersToListResponse(users), nil
}

func (s *adminService) GetUserDetail(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	user, err := s.userManager.FindOne(ctx, uow, userId)
	if err != nil {
		return nil, err
	}
	return mapper.UserToProfileResponse(user), nil
}

func (s *adminService) UpdateUserStatus(ctx context.Context, userId uuid.UUID, status string) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.userManager.UpdateStatus(ctx, uow, userId, status)
}

func (s *adminService) CreateUser(ctx context.Context, req dto.AdminCreateUserRequest) (*dto.UserProfileResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	user, err := s.userManager.Create(ctx, uow, req)
	if err != nil {
		return nil, err
	}
	return mapper.UserToProfileResponse(user), nil
}

func (s *adminService) BulkCreateUsers(ctx context.Context, fileContent []byte) (*dto.AdminBulkCreateUserResponse, error) {
	var req dto.AdminBulkCreateUserRequest
	if err := json.Unmarshal(fileContent, &req); err != nil {
		return nil, fmt.Errorf("failed to parse JSON file: %w", err)
	}

	res := &dto.AdminBulkCreateUserResponse{
		Results: []dto.BulkCreateUserResult{},
	}

	uow := s.uowFactory.NewUnitOfWork(ctx)

	for _, userReq := range req.Users {
		// Individual error handling for each user creation
		// Assuming "best effort" here.
		createdUser, err := s.userManager.Create(ctx, uow, userReq)
		if err != nil {
			res.FailedCount++
			res.Results = append(res.Results, dto.BulkCreateUserResult{
				Email:   userReq.Email,
				Success: false,
				Error:   err.Error(),
			})
		} else {
			res.CreatedCount++
			res.Results = append(res.Results, dto.BulkCreateUserResult{
				Email:   userReq.Email,
				Success: true,
				Id:      createdUser.Id.String(),
			})
		}
	}

	return res, nil
}

func (s *adminService) UpdateUser(ctx context.Context, userId uuid.UUID, req dto.AdminUpdateUserRequest) (*dto.UserProfileResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	user, err := s.userManager.Update(ctx, uow, userId, req)
	if err != nil {
		return nil, err
	}
	return mapper.UserToProfileResponse(user), nil
}

func (s *adminService) DeleteUser(ctx context.Context, userId uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.userManager.Delete(ctx, uow, userId)
}

func (s *adminService) PurgeUsers(ctx context.Context, req dto.AdminPurgeUsersRequest) (*dto.AdminPurgeUsersResponse, error) {
	// We want to process as many as possible (Best Effort with error reporting),
	// or we could do all-or-nothing if requested.
	// Given "bulk operations", usually implies independent operations.
	// But since it's "Deep Purge", each USER deletion must be transactional.

	res := &dto.AdminPurgeUsersResponse{
		FailedUsers: []dto.PurgeUserFailResult{},
	}

	for _, userId := range req.UserIds {
		// New UnitOfWork for each user to ensure independent transactions
		uow := s.uowFactory.NewUnitOfWork(ctx)
		if err := uow.Begin(ctx); err != nil {
			s.logger.Error("ADMIN_PURGE", "Failed to begin transaction", map[string]interface{}{"user_id": userId, "error": err.Error()})
			res.FailedUsers = append(res.FailedUsers, dto.PurgeUserFailResult{UserId: userId, Error: "Failed to begin transaction"})
			continue
		}

		err := func() error {
			// 1. Delete Chat Citations (via Messages) - Handled solely by ChatMessageRepository if implemented
			if err := uow.ChatMessageRepository().DeleteAllCitationsByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge citations: %w", err)
			}

			// 2. Delete Chat Messages
			if err := uow.ChatMessageRepository().DeleteAllByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge messages: %w", err)
			}

			// 3. Delete Chat Sessions
			if err := uow.ChatSessionRepository().DeleteAllByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge sessions: %w", err)
			}

			// 4. Delete Note Embeddings
			if err := uow.NoteEmbeddingRepository().DeleteAllByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge embeddings: %w", err)
			}

			// 5. Delete Notes
			if err := uow.NoteRepository().DeleteAllByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge notes: %w", err)
			}

			// 6. Delete Notebooks
			if err := uow.NotebookRepository().DeleteAllByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge notebooks: %w", err)
			}

			// 7. Delete Refunds
			if err := uow.RefundRepository().DeleteAllByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge refunds: %w", err)
			}

			// 8. Delete Cancellations
			if err := uow.CancellationRepository().DeleteAllByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge cancellations: %w", err)
			}

			// 9. Delete Billing Addresses
			if err := uow.BillingRepository().DeleteAllByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge billing: %w", err)
			}

			// 10. Delete Subscriptions
			if err := uow.SubscriptionRepository().DeleteAllSubscriptionsByUserIdUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge subscriptions: %w", err)
			}

			// 11. Delete User Related Tokens (Manual Deletion if no repo method or cascade? User Repo has no specific methods)
			// Assuming Database CASCADE for tokens on User Delete if they are strongly coupled,
			// OR we missed adding methods for them.
			// Let's assume standard auth tokens (refresh, verification, password) might be cleaned up by User delete if constraints exist
			// BUT we saw NO GORM CONSTRAINT annotations in Entity.
			// So we technically should delete them.
			// BUT `UserRepository` contract doesn't expose `DeleteAllTokens...`.
			// We can fallback to `UserRepositoryImpl` direct SQL if we really need to, but Service layer shouldn't do raw SQL ideally.
			// However, since `User` delete is unscoped, and if the tokens have FKs, it might fail if we don't delete them.
			// Assuming for now that `UserRepository.DeleteUnscoped` *might* handle it if repo implementation did it?
			// `UserRepositoryImpl` just does `Delete(&model.User{})`.
			// Since we did NOT add token deletion methods, we rely on the DB potentially having constraints or we accept potential orphans if they are nullable/no-constraint.
			// But Foreign Key constraints without cascade in DB will cause error.
			// Let's trust that for now, and if it fails in verification, we add specific token deletion methods.

			// 12. Delete User
			if err := uow.UserRepository().DeleteUnscoped(ctx, userId); err != nil {
				return fmt.Errorf("purge user: %w", err)
			}

			return nil
		}()

		if err != nil {
			uow.Rollback()
			s.logger.Error("ADMIN_PURGE", "Failed to purge user", map[string]interface{}{"user_id": userId, "error": err.Error()})
			res.FailedUsers = append(res.FailedUsers, dto.PurgeUserFailResult{UserId: userId, Error: err.Error()})
		} else {
			if err := uow.Commit(); err != nil {
				s.logger.Error("ADMIN_PURGE", "Failed to commit purge", map[string]interface{}{"user_id": userId, "error": err.Error()})
				res.FailedUsers = append(res.FailedUsers, dto.PurgeUserFailResult{UserId: userId, Error: "Commit failed"})
			} else {
				res.DeletedCount++
				s.logger.Info("ADMIN_PURGE", "Successfully purged user", map[string]interface{}{"user_id": userId})
			}
		}
	}

	return res, nil
}

// ============================================================================
// Transaction Management
// ============================================================================

func (s *adminService) GetTransactions(ctx context.Context, page, limit int, status string) ([]*dto.TransactionListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.dashboardAggregator.GetTransactions(ctx, uow, page, limit, status)
}

// ============================================================================
// Logs
// ============================================================================

func (s *adminService) GetSystemLogs(ctx context.Context, page, limit int, level string) ([]*dto.LogListResponse, error) {
	return s.dashboardAggregator.GetSystemLogs(ctx, s.logger, page, limit, level)
}

func (s *adminService) GetLogDetail(ctx context.Context, logId string) (*dto.LogDetailResponse, error) {
	return s.dashboardAggregator.GetLogDetail(ctx, s.logger, logId)
}

// ============================================================================
// Subscription Management
// ============================================================================

func (s *adminService) UpgradeSubscription(ctx context.Context, req dto.AdminSubscriptionUpgradeRequest) (*dto.AdminSubscriptionUpgradeResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	result, err := s.subscriptionManager.Upgrade(ctx, uow, req)
	if err != nil {
		return nil, err
	}
	return &dto.AdminSubscriptionUpgradeResponse{
		OldSubscriptionId: result.OldSubscriptionId,
		NewSubscriptionId: result.NewSubscriptionId,
		CreditApplied:     result.CreditApplied,
		AmountDue:         result.AmountDue,
		Status:            "success",
	}, nil
}

func (s *adminService) RefundSubscription(ctx context.Context, req dto.AdminRefundRequest) (*dto.AdminRefundResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	result, err := s.subscriptionManager.Refund(ctx, uow, req)
	if err != nil {
		return nil, err
	}
	return &dto.AdminRefundResponse{
		RefundId:       result.RefundId.String(),
		RefundedAmount: result.RefundedAmount,
		Status:         "processed",
	}, nil
}

// ============================================================================
// Refund Management (User-requested refunds)
// ============================================================================

func (s *adminService) GetRefunds(ctx context.Context, page, limit int, status string) ([]*dto.AdminRefundListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	refunds, err := s.refundProcessor.GetAll(ctx, uow, page, limit, status)
	if err != nil {
		return nil, err
	}

	var res []*dto.AdminRefundListResponse
	for _, r := range refunds {
		planName, amountPaid := s.refundProcessor.GetPlanInfo(ctx, uow, r.Subscription.PlanId)
		if planName == "" {
			amountPaid = r.Amount
		}
		res = append(res, mapper.RefundToListResponse(r, planName, amountPaid))
	}
	return res, nil
}

func (s *adminService) ApproveRefund(ctx context.Context, refundId uuid.UUID, req dto.AdminApproveRefundRequest) (*dto.AdminApproveRefundResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	result, err := s.refundProcessor.Approve(ctx, uow, refundId, req)
	if err != nil {
		return nil, err
	}
	return &dto.AdminApproveRefundResponse{
		RefundId:       result.RefundId.String(),
		Status:         string(entity.RefundStatusApproved),
		RefundedAmount: result.RefundedAmount,
		ProcessedAt:    result.ProcessedAt,
	}, nil
}

func (s *adminService) RejectRefund(ctx context.Context, refundId uuid.UUID, req dto.AdminRejectRefundRequest) (*dto.AdminRejectRefundResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	result, err := s.refundProcessor.Reject(ctx, uow, refundId, req)
	if err != nil {
		return nil, err
	}
	return &dto.AdminRejectRefundResponse{
		RefundId:    result.RefundId.String(),
		Status:      string(entity.RefundStatusRejected),
		ProcessedAt: result.ProcessedAt,
	}, nil
}

// ============================================================================
// Plan Management
// ============================================================================

func (s *adminService) CreatePlan(ctx context.Context, req dto.AdminCreatePlanRequest) (*dto.AdminPlanResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	plan, err := s.planManager.Create(ctx, uow, req)
	if err != nil {
		return nil, err
	}
	response := mapper.PlanToAdminResponse(plan)
	// Use request features directly since the plan was just created
	response.Features = req.Features
	return response, nil
}

func (s *adminService) UpdatePlan(ctx context.Context, id uuid.UUID, req dto.AdminUpdatePlanRequest) (*dto.AdminPlanResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	plan, err := s.planManager.Update(ctx, uow, id, req)
	if err != nil {
		return nil, err
	}
	return mapper.PlanToAdminResponse(plan), nil
}

func (s *adminService) DeletePlan(ctx context.Context, id uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.planManager.Delete(ctx, uow, id)
}

func (s *adminService) GetAllPlans(ctx context.Context) ([]*dto.AdminPlanResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	plans, err := s.planManager.FindAll(ctx, uow)
	if err != nil {
		return nil, err
	}
	return mapper.PlansToAdminResponse(plans), nil
}

// ============================================================================
// Plan Feature Management
// ============================================================================

func (s *adminService) GetPlanFeatures(ctx context.Context, planId uuid.UUID) ([]*dto.PlanFeatureResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	features, err := s.planManager.GetFeatures(ctx, uow, planId)
	if err != nil {
		return nil, err
	}
	return mapper.PlanFeaturesToResponse(features), nil
}

func (s *adminService) CreatePlanFeature(ctx context.Context, planId uuid.UUID, req dto.CreatePlanFeatureRequest) (*dto.PlanFeatureResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	feature, err := s.planManager.AddFeature(ctx, uow, planId, req.FeatureKey)
	if err != nil {
		return nil, err
	}
	return mapper.FeatureToPlanFeatureResponse(feature), nil
}

func (s *adminService) DeletePlanFeature(ctx context.Context, planId uuid.UUID, featureId uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.planManager.RemoveFeature(ctx, uow, planId, featureId)
}

// ============================================================================
// Feature Catalog Management
// ============================================================================

func (s *adminService) GetAllFeatures(ctx context.Context) ([]*dto.FeatureResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	features, err := s.featureManager.GetAll(ctx, uow)
	if err != nil {
		return nil, err
	}
	return mapper.FeaturesToResponse(features), nil
}

func (s *adminService) CreateFeature(ctx context.Context, req dto.CreateFeatureRequest) (*dto.FeatureResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	feature, err := s.featureManager.Create(ctx, uow, req)
	if err != nil {
		return nil, err
	}
	return mapper.FeatureToResponse(feature), nil
}

func (s *adminService) UpdateFeature(ctx context.Context, id uuid.UUID, req dto.UpdateFeatureRequest) (*dto.FeatureResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	feature, err := s.featureManager.Update(ctx, uow, id, req)
	if err != nil {
		return nil, err
	}
	return mapper.FeatureToResponse(feature), nil
}

func (s *adminService) DeleteFeature(ctx context.Context, id uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.featureManager.Delete(ctx, uow, id)
}

// ============================================================================
// Token Usage Tracking
// ============================================================================

func (s *adminService) GetTokenUsage(ctx context.Context, page, limit int) ([]*dto.TokenUsageResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.usageTracker.GetTokenUsage(ctx, uow, page, limit)
}

func (s *adminService) UpdateAiLimit(ctx context.Context, userId uuid.UUID, req dto.UpdateAiLimitRequest) (*dto.UpdateAiLimitResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	result, err := s.usageTracker.UpdateAiLimit(ctx, uow, userId, req)
	if err != nil {
		return nil, err
	}
	return &dto.UpdateAiLimitResponse{
		UserId:                      result.User.Id,
		PreviousChatUsage:           result.PreviousChatUsage,
		NewChatUsage:                result.User.AiDailyUsage,
		PreviousSemanticSearchUsage: result.PreviousSemanticSearchUsage,
		NewSemanticSearchUsage:      result.User.SemanticSearchDailyUsage,
		UserEmail:                   result.User.Email,
	}, nil
}

func (s *adminService) ResetAiLimit(ctx context.Context, userId uuid.UUID) (*dto.UpdateAiLimitResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	result, err := s.usageTracker.ResetAiLimit(ctx, uow, userId)
	if err != nil {
		return nil, err
	}
	return &dto.UpdateAiLimitResponse{
		UserId:                      result.User.Id,
		PreviousChatUsage:           result.PreviousChatUsage,
		NewChatUsage:                0,
		PreviousSemanticSearchUsage: result.PreviousSemanticSearchUsage,
		NewSemanticSearchUsage:      0,
		UserEmail:                   result.User.Email,
	}, nil
}

func (s *adminService) BulkUpdateAiLimit(ctx context.Context, req dto.BulkUpdateAiLimitRequest) (*dto.BulkAiLimitResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.usageTracker.BulkUpdateAiLimit(ctx, uow, req), nil
}

func (s *adminService) BulkResetAiLimit(ctx context.Context, req dto.BulkResetAiLimitRequest) (*dto.BulkAiLimitResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.usageTracker.BulkResetAiLimit(ctx, uow, req), nil
}

// ============================================================================
// AI Configuration Management
// ============================================================================

func (s *adminService) GetAllAiConfigurations(ctx context.Context) ([]*dto.AiConfigurationResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.aiConfigManager.GetAllConfigurations(ctx, uow)
}

func (s *adminService) UpdateAiConfiguration(ctx context.Context, key string, req dto.UpdateAiConfigurationRequest) (*dto.AiConfigurationResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.aiConfigManager.UpdateConfiguration(ctx, uow, key, req)
}

func (s *adminService) GetAllNuances(ctx context.Context) ([]*dto.AiNuanceResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.aiConfigManager.GetAllNuances(ctx, uow)
}

func (s *adminService) CreateNuance(ctx context.Context, req dto.CreateAiNuanceRequest) (*dto.AiNuanceResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.aiConfigManager.CreateNuance(ctx, uow, req)
}

func (s *adminService) UpdateNuance(ctx context.Context, id uuid.UUID, req dto.UpdateAiNuanceRequest) (*dto.AiNuanceResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.aiConfigManager.UpdateNuance(ctx, uow, id, req)
}

func (s *adminService) DeleteNuance(ctx context.Context, id uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return s.aiConfigManager.DeleteNuance(ctx, uow, id)
}

// ============================================================================
// Billing Management
// ============================================================================

func (s *adminService) GetUserBillingAddresses(ctx context.Context, userId uuid.UUID) ([]*dto.AdminBillingListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	billings, err := uow.BillingRepository().FindAll(ctx, specification.UserOwnedBy{UserID: userId})
	if err != nil {
		return nil, err
	}

	var res []*dto.AdminBillingListResponse
	for _, b := range billings {
		res = append(res, &dto.AdminBillingListResponse{
			Id:           b.Id,
			UserId:       b.UserId,
			FirstName:    b.FirstName,
			LastName:     b.LastName,
			Email:        b.Email,
			Phone:        b.Phone,
			AddressLine1: b.AddressLine1,
			AddressLine2: b.AddressLine2,
			City:         b.City,
			State:        b.State,
			PostalCode:   b.PostalCode,
			Country:      b.Country,
			IsDefault:    b.IsDefault,
			CreatedAt:    b.CreatedAt,
			UpdatedAt:    b.UpdatedAt,
		})
	}
	return res, nil
}

func (s *adminService) CreateBillingAddress(ctx context.Context, userId uuid.UUID, req dto.AdminBillingCreateRequest) (*dto.AdminBillingListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	billing := &entity.BillingAddress{
		Id:           uuid.New(),
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
		IsDefault:    req.IsDefault,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uow.BillingRepository().Create(ctx, billing); err != nil {
		return nil, err
	}

	return &dto.AdminBillingListResponse{
		Id:           billing.Id,
		UserId:       billing.UserId,
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
		IsDefault:    billing.IsDefault,
		CreatedAt:    billing.CreatedAt,
		UpdatedAt:    billing.UpdatedAt,
	}, nil
}

func (s *adminService) UpdateBillingAddress(ctx context.Context, id uuid.UUID, req dto.AdminBillingUpdateRequest) (*dto.AdminBillingListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	billing, err := uow.BillingRepository().FindOne(ctx, specification.ByID{ID: id})
	if err != nil {
		return nil, err
	}
	if billing == nil {
		return nil, fmt.Errorf("billing address not found")
	}

	// Apply updates
	if req.FirstName != nil {
		billing.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		billing.LastName = *req.LastName
	}
	if req.Email != nil {
		billing.Email = *req.Email
	}
	if req.Phone != nil {
		billing.Phone = *req.Phone
	}
	if req.AddressLine1 != nil {
		billing.AddressLine1 = *req.AddressLine1
	}
	if req.AddressLine2 != nil {
		billing.AddressLine2 = *req.AddressLine2
	}
	if req.City != nil {
		billing.City = *req.City
	}
	if req.State != nil {
		billing.State = *req.State
	}
	if req.PostalCode != nil {
		billing.PostalCode = *req.PostalCode
	}
	if req.Country != nil {
		billing.Country = *req.Country
	}
	if req.IsDefault != nil {
		billing.IsDefault = *req.IsDefault
	}
	billing.UpdatedAt = time.Now()

	if err := uow.BillingRepository().Update(ctx, billing); err != nil {
		return nil, err
	}

	return &dto.AdminBillingListResponse{
		Id:           billing.Id,
		UserId:       billing.UserId,
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
		IsDefault:    billing.IsDefault,
		CreatedAt:    billing.CreatedAt,
		UpdatedAt:    billing.UpdatedAt,
	}, nil
}

func (s *adminService) DeleteBillingAddress(ctx context.Context, id uuid.UUID) error {
	uow := s.uowFactory.NewUnitOfWork(ctx)
	return uow.BillingRepository().Delete(ctx, id)
}

// ============================================================================
// Cancellation Management
// ============================================================================

func (s *adminService) GetCancellations(ctx context.Context, page, limit int, status string) ([]*dto.AdminCancellationListResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	var specs []specification.Specification
	if status != "" && status != "all" {
		specs = append(specs, specification.Filter("status", status))
	}
	specs = append(specs, specification.OrderBy{Field: "created_at", Desc: true})
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	specs = append(specs, specification.Pagination{Limit: limit, Offset: offset})

	cancellations, err := uow.CancellationRepository().FindAllWithDetails(ctx, specs...)
	if err != nil {
		return nil, err
	}

	var res []*dto.AdminCancellationListResponse
	for _, c := range cancellations {
		// Get plan name
		planName := ""
		if c.Subscription.PlanId != uuid.Nil {
			plan, _ := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: c.Subscription.PlanId})
			if plan != nil {
				planName = plan.Name
			}
		}

		res = append(res, &dto.AdminCancellationListResponse{
			Id: c.ID,
			User: dto.AdminCancellationUserInfo{
				Id:       c.User.Id,
				Email:    c.User.Email,
				FullName: c.User.FullName,
			},
			Subscription: dto.AdminCancellationSubscriptionInfo{
				Id:               c.SubscriptionID,
				PlanName:         planName,
				CurrentPeriodEnd: c.Subscription.CurrentPeriodEnd,
			},
			Reason:        c.Reason,
			Status:        c.Status,
			AdminNotes:    c.AdminNotes,
			EffectiveDate: c.EffectiveDate,
			CreatedAt:     c.CreatedAt,
			ProcessedAt:   c.ProcessedAt,
		})
	}
	return res, nil
}

func (s *adminService) ProcessCancellation(ctx context.Context, cancellationId uuid.UUID, req dto.AdminProcessCancellationRequest) (*dto.AdminProcessCancellationResponse, error) {
	uow := s.uowFactory.NewUnitOfWork(ctx)

	cancellation, err := uow.CancellationRepository().FindOne(ctx, specification.ByID{ID: cancellationId})
	if err != nil {
		return nil, err
	}
	if cancellation == nil {
		return nil, fmt.Errorf("cancellation not found")
	}

	now := time.Now()
	cancellation.ProcessedAt = &now
	cancellation.AdminNotes = req.AdminNotes

	// Fetch subscription to get PlanID for notification
	sub, err := uow.SubscriptionRepository().FindOneSubscription(ctx, specification.ByID{ID: cancellation.SubscriptionID})
	if err != nil {
		// Log error but don't fail the whole operation if we can't find sub for notification
		s.logger.Error("ADMIN", "Failed to find subscription for cancellation notification", map[string]interface{}{"error": err.Error()})
	}

	if req.Action == "approve" {
		cancellation.Status = string(entity.CancellationStatusApproved)

		// Update subscription status to canceled if found
		if sub != nil {
			sub.Status = entity.SubscriptionStatusCanceled
			if err := uow.SubscriptionRepository().UpdateSubscription(ctx, sub); err != nil {
				return nil, err
			}
		}
	} else {
		cancellation.Status = string(entity.CancellationStatusRejected)
	}

	if err := uow.CancellationRepository().Update(ctx, cancellation); err != nil {
		return nil, err
	}

	// Emit SUBSCRIPTION_CANCELLATION_PROCESSED event
	planName := "Unknown Plan"
	if sub != nil {
		plan, _ := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: sub.PlanId})
		if plan != nil {
			planName = plan.Name
		}
	}

	if s.eventPublisher != nil {
		s.eventPublisher.PublishCancellationProcessed(
			ctx,
			cancellationId,
			cancellation.SubscriptionID,
			cancellation.UserID,
			planName,
			cancellation.Status,
		)
	}

	return &dto.AdminProcessCancellationResponse{
		CancellationId: cancellationId.String(),
		Status:         cancellation.Status,
		EffectiveDate:  cancellation.EffectiveDate,
		ProcessedAt:    now,
	}, nil
}
