package mapper

import (
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
)

// UserToListResponse converts entity to list response DTO
func UserToListResponse(u *entity.User) *dto.UserListResponse {
	if u == nil {
		return nil
	}
	return &dto.UserListResponse{
		Id:        u.Id,
		Email:     u.Email,
		FullName:  u.FullName,
		Role:      string(u.Role),
		Status:    string(u.Status),
		CreatedAt: u.CreatedAt,
	}
}

// UsersToListResponse converts multiple entities to list response DTOs
func UsersToListResponse(users []*entity.User) []*dto.UserListResponse {
	var res []*dto.UserListResponse
	for _, u := range users {
		res = append(res, UserToListResponse(u))
	}
	return res
}

// UserToProfileResponse converts entity to profile response DTO
func UserToProfileResponse(u *entity.User) *dto.UserProfileResponse {
	if u == nil {
		return nil
	}
	return &dto.UserProfileResponse{
		Id:           u.Id,
		Email:        u.Email,
		FullName:     u.FullName,
		Role:         string(u.Role),
		Status:       string(u.Status),
		AiDailyUsage: u.AiDailyUsage,
		CreatedAt:    u.CreatedAt,
	}
}

// PlanToAdminResponse converts entity to admin plan response DTO
func PlanToAdminResponse(p *entity.SubscriptionPlan) *dto.AdminPlanResponse {
	if p == nil {
		return nil
	}
	return &dto.AdminPlanResponse{
		Id:            p.Id,
		Name:          p.Name,
		Slug:          p.Slug,
		Description:   p.Description,
		Tagline:       p.Tagline,
		Price:         p.Price,
		TaxRate:       p.TaxRate,
		BillingPeriod: string(p.BillingPeriod),
		IsMostPopular: p.IsMostPopular,
		IsActive:      p.IsActive,
		SortOrder:     p.SortOrder,
		Features: dto.PlanFeaturesDTO{
			MaxNotebooks:             p.MaxNotebooks,
			MaxNotesPerNotebook:      p.MaxNotesPerNotebook,
			SemanticSearch:           p.SemanticSearchEnabled,
			AiChat:                   p.AiChatEnabled,
			AiChatDailyLimit:         p.AiChatDailyLimit,
			SemanticSearchDailyLimit: p.SemanticSearchDailyLimit,
		},
	}
}

// PlansToAdminResponse converts multiple entities to admin plan response DTOs
func PlansToAdminResponse(plans []*entity.SubscriptionPlan) []*dto.AdminPlanResponse {
	var res []*dto.AdminPlanResponse
	for _, p := range plans {
		res = append(res, PlanToAdminResponse(p))
	}
	return res
}

// FeatureToResponse converts entity to feature response DTO
func FeatureToResponse(f *entity.Feature) *dto.FeatureResponse {
	if f == nil {
		return nil
	}
	return &dto.FeatureResponse{
		Id:          f.Id,
		Key:         f.Key,
		Name:        f.Name,
		Description: f.Description,
		Category:    f.Category,
		IsActive:    f.IsActive,
		SortOrder:   f.SortOrder,
	}
}

// FeaturesToResponse converts multiple entities to feature response DTOs
func FeaturesToResponse(features []*entity.Feature) []*dto.FeatureResponse {
	var res []*dto.FeatureResponse
	for _, f := range features {
		res = append(res, FeatureToResponse(f))
	}
	return res
}

// FeatureToPlanFeatureResponse converts entity to plan feature response DTO
func FeatureToPlanFeatureResponse(f *entity.Feature) *dto.PlanFeatureResponse {
	if f == nil {
		return nil
	}
	return &dto.PlanFeatureResponse{
		Id:          f.Id,
		Key:         f.Key,
		Name:        f.Name,
		Description: f.Description,
	}
}

// PlanFeaturesToResponse converts plan features to response DTOs
func PlanFeaturesToResponse(features []entity.Feature) []*dto.PlanFeatureResponse {
	var res []*dto.PlanFeatureResponse
	for _, f := range features {
		res = append(res, &dto.PlanFeatureResponse{
			Id:          f.Id,
			Key:         f.Key,
			Name:        f.Name,
			Description: f.Description,
		})
	}
	return res
}

// TransactionToListResponse converts transaction entity to list response DTO
func TransactionToListResponse(t *entity.SubscriptionTransaction) *dto.TransactionListResponse {
	if t == nil {
		return nil
	}
	return &dto.TransactionListResponse{
		Id:              t.Id,
		UserId:          t.UserId,
		UserEmail:       t.UserEmail,
		PlanName:        t.PlanName,
		Amount:          t.Amount,
		Status:          string(t.Status),
		PaymentStatus:   string(t.PaymentStatus),
		TransactionDate: t.CreatedAt,
		MidtransOrderId: t.MidtransOrderId,
	}
}

// TransactionsToListResponse converts multiple transaction entities to list response DTOs
func TransactionsToListResponse(txs []*entity.SubscriptionTransaction) []*dto.TransactionListResponse {
	var res []*dto.TransactionListResponse
	for _, t := range txs {
		res = append(res, TransactionToListResponse(t))
	}
	return res
}

// LogToListResponse converts log entry to list response DTO
func LogToListResponse(l interface {
	GetId() string
	GetLevel() string
	GetModule() string
	GetMessage() string
	GetTimestamp() string
}) *dto.LogListResponse {
	ts, _ := time.Parse(time.RFC3339, l.GetTimestamp())
	return &dto.LogListResponse{
		Id:        l.GetId(),
		Level:     l.GetLevel(),
		Module:    l.GetModule(),
		Message:   l.GetMessage(),
		CreatedAt: ts,
	}
}

// RefundToListResponse converts refund entity to admin list response
func RefundToListResponse(r *entity.Refund, planName string, amountPaid float64) *dto.AdminRefundListResponse {
	if r == nil {
		return nil
	}
	return &dto.AdminRefundListResponse{
		Id: r.ID,
		User: dto.AdminRefundUserInfo{
			Id:       r.User.Id,
			Email:    r.User.Email,
			FullName: r.User.FullName,
		},
		Subscription: dto.AdminRefundSubscriptionInfo{
			Id:          r.Subscription.Id,
			PlanName:    planName,
			AmountPaid:  amountPaid,
			PaymentDate: r.Subscription.CurrentPeriodStart,
		},
		Amount:      r.Amount,
		Reason:      r.Reason,
		Status:      r.Status,
		AdminNotes:  r.AdminNotes,
		CreatedAt:   r.CreatedAt,
		ProcessedAt: r.ProcessedAt,
	}
}

// TokenUsageToResponse builds token usage DTO from user and plan info
func TokenUsageToResponse(user *entity.User, planName string, chatLimit, searchLimit int) *dto.TokenUsageResponse {
	if user == nil {
		return nil
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

	return &dto.TokenUsageResponse{
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
	}
}
