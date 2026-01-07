package usage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/repository/specification"
	"ai-notetaking-be/internal/repository/unitofwork"
	adminEvents "ai-notetaking-be/pkg/admin/events"

	"github.com/google/uuid"
)

// UpdateResult contains update operation results
type UpdateResult struct {
	User                        *entity.User
	PreviousChatUsage           int
	PreviousSemanticSearchUsage int
}

// Tracker handles AI usage tracking operations
type Tracker struct {
	logger    logger.ILogger
	publisher adminEvents.Publisher
}

// NewTracker creates a new usage tracker
func NewTracker(logger logger.ILogger, publisher adminEvents.Publisher) *Tracker {
	return &Tracker{
		logger:    logger,
		publisher: publisher,
	}
}

// GetTokenUsage retrieves paginated users with their AI token usage
func (t *Tracker) GetTokenUsage(ctx context.Context, uow unitofwork.UnitOfWork, page, limit int) ([]*dto.TokenUsageResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	// Get users ordered by AI usage (highest first)
	users, err := uow.UserRepository().FindAll(ctx,
		specification.Pagination{Limit: limit, Offset: offset},
		specification.OrderBy{Field: "ai_daily_usage", Desc: true},
	)
	if err != nil {
		return nil, err
	}

	var res []*dto.TokenUsageResponse
	for _, user := range users {
		// Determine Plan Limits
		chatLimit := 0
		searchLimit := 0
		planName := "No Plan"

		subs, err := uow.SubscriptionRepository().FindAllSubscriptions(ctx,
			specification.UserOwnedBy{UserID: user.Id},
			specification.Filter("status", "active"),
		)

		if err == nil && len(subs) > 0 {
			// Get the plan for this subscription
			plan, err := uow.SubscriptionRepository().FindOnePlan(ctx, specification.ByID{ID: subs[0].PlanId})
			if err == nil && plan != nil {
				planName = plan.Name
				chatLimit = plan.AiChatDailyLimit
				searchLimit = plan.SemanticSearchDailyLimit
			}
		}

		// Apply Admin Override for Chat Limit if set
		if user.AiDailyLimitOverride != nil {
			chatLimit = *user.AiDailyLimitOverride
		}

		// Compute remaining for Chat
		chatRemaining := 0
		if chatLimit == -1 {
			chatRemaining = -1 // Unlimited
		} else if chatLimit > user.AiDailyUsage {
			chatRemaining = chatLimit - user.AiDailyUsage
		}

		// Compute remaining for Search
		searchRemaining := 0
		if searchLimit == -1 {
			searchRemaining = -1
		} else if searchLimit > user.SemanticSearchDailyUsage {
			searchRemaining = searchLimit - user.SemanticSearchDailyUsage
		}

		res = append(res, &dto.TokenUsageResponse{
			UserId:                       user.Id,
			Email:                        user.Email,
			FullName:                     user.FullName,
			PlanName:                     planName,
			AiChatDailyUsage:             user.AiDailyUsage,
			AiChatDailyLimit:             chatLimit,
			AiChatDailyRemaining:         chatRemaining,
			SemanticSearchDailyUsage:     user.SemanticSearchDailyUsage,
			SemanticSearchDailyLimit:     searchLimit,
			SemanticSearchDailyRemaining: searchRemaining,
			AiDailyUsageLastReset:        user.AiDailyUsageLastReset,
			SemanticSearchUsageLastReset: user.SemanticSearchDailyUsageLastReset,
		})
	}

	return res, nil
}

// UpdateAiLimit updates a user's AI daily usage
func (t *Tracker) UpdateAiLimit(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID, req dto.UpdateAiLimitRequest) (*UpdateResult, error) {
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	previousChatUsage := user.AiDailyUsage
	previousSearchUsage := user.SemanticSearchDailyUsage
	updated := false
	limitDescriptions := []string{}

	// Update Chat Usage if provided
	if req.AiChatDailyUsage != nil {
		user.AiDailyUsage = *req.AiChatDailyUsage
		user.AiDailyUsageLastReset = time.Now()
		updated = true
		desc := "chat usage updated"
		if *req.AiChatDailyUsage == 0 {
			desc = "chat usage reset"
		}
		limitDescriptions = append(limitDescriptions, desc)
	}

	// Update Semantic Search Usage if provided
	if req.SemanticSearchDailyUsage != nil {
		user.SemanticSearchDailyUsage = *req.SemanticSearchDailyUsage
		user.SemanticSearchDailyUsageLastReset = time.Now()
		updated = true
		desc := "search usage updated"
		if *req.SemanticSearchDailyUsage == 0 {
			desc = "search usage reset"
		}
		limitDescriptions = append(limitDescriptions, desc)
	}

	if !updated {
		return &UpdateResult{
			User:                        user,
			PreviousChatUsage:           previousChatUsage,
			PreviousSemanticSearchUsage: previousSearchUsage,
		}, nil
	}

	if err := uow.UserRepository().Update(ctx, user); err != nil {
		return nil, err
	}

	// Emit AI_LIMIT_UPDATED event
	t.publisher.PublishAiLimitUpdated(ctx, userId, user.Email, previousChatUsage, user.AiDailyUsage, previousSearchUsage, user.SemanticSearchDailyUsage, strings.Join(limitDescriptions, ", "))

	t.logger.Info("ADMIN", "Updated AI usage", map[string]interface{}{
		"user_id":                   userId,
		"new_chat_usage":            user.AiDailyUsage,
		"new_semantic_search_usage": user.SemanticSearchDailyUsage,
	})

	return &UpdateResult{
		User:                        user,
		PreviousChatUsage:           previousChatUsage,
		PreviousSemanticSearchUsage: previousSearchUsage,
	}, nil
}

// ResetAiLimit resets a user's AI daily usages to 0
func (t *Tracker) ResetAiLimit(ctx context.Context, uow unitofwork.UnitOfWork, userId uuid.UUID) (*UpdateResult, error) {
	user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	previousChatUsage := user.AiDailyUsage
	previousSearchUsage := user.SemanticSearchDailyUsage

	// Reset BOTH
	user.AiDailyUsage = 0
	user.AiDailyUsageLastReset = time.Now()
	user.SemanticSearchDailyUsage = 0
	user.SemanticSearchDailyUsageLastReset = time.Now()

	if err := uow.UserRepository().Update(ctx, user); err != nil {
		return nil, err
	}

	t.publisher.PublishAiLimitUpdated(ctx, userId, user.Email, previousChatUsage, 0, previousSearchUsage, 0, "all usage reset")

	t.logger.Info("ADMIN", "Reset AI usage", map[string]interface{}{
		"user_id": userId,
	})

	return &UpdateResult{
		User:                        user,
		PreviousChatUsage:           previousChatUsage,
		PreviousSemanticSearchUsage: previousSearchUsage,
	}, nil
}

// BulkUpdateAiLimit updates multiple users' AI daily usage
func (t *Tracker) BulkUpdateAiLimit(ctx context.Context, uow unitofwork.UnitOfWork, req dto.BulkUpdateAiLimitRequest) *dto.BulkAiLimitResponse {
	response := &dto.BulkAiLimitResponse{
		TotalRequested: len(req.UserIds),
		TotalUpdated:   0,
		FailedUserIds:  []uuid.UUID{},
	}

	for _, userId := range req.UserIds {
		user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
		if err != nil || user == nil {
			response.FailedUserIds = append(response.FailedUserIds, userId)
			continue
		}

		previousChatUsage := user.AiDailyUsage
		previousSearchUsage := user.SemanticSearchDailyUsage
		updated := false
		limitDescriptions := []string{}

		if req.AiChatDailyUsage != nil {
			user.AiDailyUsage = *req.AiChatDailyUsage
			user.AiDailyUsageLastReset = time.Now()
			updated = true
			desc := "chat usage updated"
			if *req.AiChatDailyUsage == 0 {
				desc = "chat usage reset"
			}
			limitDescriptions = append(limitDescriptions, desc)
		}

		if req.SemanticSearchDailyUsage != nil {
			user.SemanticSearchDailyUsage = *req.SemanticSearchDailyUsage
			user.SemanticSearchDailyUsageLastReset = time.Now()
			updated = true
			desc := "search usage updated"
			if *req.SemanticSearchDailyUsage == 0 {
				desc = "search usage reset"
			}
			limitDescriptions = append(limitDescriptions, desc)
		}

		if !updated {
			response.FailedUserIds = append(response.FailedUserIds, userId)
			continue
		}

		if err := uow.UserRepository().Update(ctx, user); err != nil {
			response.FailedUserIds = append(response.FailedUserIds, userId)
			continue
		}

		// Emit event per user
		t.publisher.PublishAiLimitUpdated(ctx, userId, user.Email, previousChatUsage, user.AiDailyUsage, previousSearchUsage, user.SemanticSearchDailyUsage, strings.Join(limitDescriptions, ", "))

		response.TotalUpdated++
	}

	t.logger.Info("ADMIN", "Bulk updated AI usage", map[string]interface{}{
		"total_requested": len(req.UserIds),
		"total_updated":   response.TotalUpdated,
	})

	return response
}

// BulkResetAiLimit resets multiple users' AI daily usage to 0
func (t *Tracker) BulkResetAiLimit(ctx context.Context, uow unitofwork.UnitOfWork, req dto.BulkResetAiLimitRequest) *dto.BulkAiLimitResponse {
	response := &dto.BulkAiLimitResponse{
		TotalRequested: len(req.UserIds),
		TotalUpdated:   0,
		FailedUserIds:  []uuid.UUID{},
	}

	for _, userId := range req.UserIds {
		user, err := uow.UserRepository().FindOne(ctx, specification.ByID{ID: userId})
		if err != nil || user == nil {
			response.FailedUserIds = append(response.FailedUserIds, userId)
			continue
		}

		previousChatUsage := user.AiDailyUsage
		previousSearchUsage := user.SemanticSearchDailyUsage

		user.AiDailyUsage = 0
		user.AiDailyUsageLastReset = time.Now()
		user.SemanticSearchDailyUsage = 0
		user.SemanticSearchDailyUsageLastReset = time.Now()

		if err := uow.UserRepository().Update(ctx, user); err != nil {
			response.FailedUserIds = append(response.FailedUserIds, userId)
			continue
		}

		t.publisher.PublishAiLimitUpdated(ctx, userId, user.Email, previousChatUsage, 0, previousSearchUsage, 0, "all usage reset")

		response.TotalUpdated++
	}

	t.logger.Info("ADMIN", "Bulk reset AI usage", map[string]interface{}{
		"total_requested": len(req.UserIds),
		"total_updated":   response.TotalUpdated,
	})

	return response
}
