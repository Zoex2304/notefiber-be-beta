package dto

import (
	"time"

	"github.com/google/uuid"
)

// --- User Management ---

type AdminUserListRequest struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Search string `query:"search"`
	Role   string `query:"role"`
	Status string `query:"status"`
}

type TokenUsageResponse struct {
	UserId                       uuid.UUID `json:"user_id"`
	Email                        string    `json:"email"`
	FullName                     string    `json:"full_name"`
	PlanName                     string    `json:"plan_name"`
	AiChatDailyUsage             int       `json:"ai_chat_daily_usage"`
	AiChatDailyLimit             int       `json:"ai_chat_daily_limit"`
	AiChatDailyRemaining         int       `json:"ai_chat_daily_remaining"`
	SemanticSearchDailyUsage     int       `json:"semantic_search_daily_usage"`
	SemanticSearchDailyLimit     int       `json:"semantic_search_daily_limit"`
	SemanticSearchDailyRemaining int       `json:"semantic_search_daily_remaining"`
	AiDailyUsageLastReset        time.Time `json:"ai_daily_usage_last_reset"`
	SemanticSearchUsageLastReset time.Time `json:"semantic_search_usage_last_reset"`
}

type UserListResponse struct {
	Id        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active pending banned"`
	Reason string `json:"reason,omitempty"`
}

type AdminCreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required"`
	Role     string `json:"role" validate:"required,oneof=user admin"`
}

type AdminBulkCreateUserRequest struct {
	Users []AdminCreateUserRequest `json:"users" validate:"required,min=1"`
}

type AdminBulkCreateUserResponse struct {
	CreatedCount int                    `json:"created_count"`
	FailedCount  int                    `json:"failed_count"`
	Results      []BulkCreateUserResult `json:"results"`
}

type BulkCreateUserResult struct {
	Email   string `json:"email"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Id      string `json:"id,omitempty"` // User ID if success
}

type AdminUpdateUserRequest struct {
	FullName             string `json:"full_name"`
	Email                string `json:"email" validate:"omitempty,email"`
	Role                 string `json:"role" validate:"omitempty,oneof=user admin"`
	Status               string `json:"status" validate:"omitempty,oneof=active pending banned"`
	Avatar               string `json:"avatar"`
	AiDailyLimitOverride *int   `json:"ai_daily_limit_override"`
}

type AdminPurgeUsersRequest struct {
	UserIds []uuid.UUID `json:"user_ids" validate:"required,min=1"`
}

type AdminPurgeUsersResponse struct {
	DeletedCount int                   `json:"deleted_count"`
	FailedUsers  []PurgeUserFailResult `json:"failed_users,omitempty"`
}

type PurgeUserFailResult struct {
	UserId uuid.UUID `json:"user_id"`
	Error  string    `json:"error"`
}

// --- Subscription Management ---

type AdminSubscriptionUpgradeRequest struct {
	UserId    uuid.UUID `json:"user_id" validate:"required"`
	NewPlanId uuid.UUID `json:"new_plan_id" validate:"required"`
}

type AdminSubscriptionUpgradeResponse struct {
	OldSubscriptionId uuid.UUID `json:"old_subscription_id"`
	NewSubscriptionId uuid.UUID `json:"new_subscription_id"`
	CreditApplied     float64   `json:"credit_applied"`
	AmountDue         float64   `json:"amount_due"`
	Status            string    `json:"status"`
}

type AdminRefundRequest struct {
	SubscriptionId uuid.UUID `json:"subscription_id" validate:"required"`
	Reason         string    `json:"reason" validate:"required"`
	Amount         *float64  `json:"amount,omitempty"` // If nil, full refund
}

type AdminRefundResponse struct {
	RefundId       string  `json:"refund_id"` // Transaction ID
	RefundedAmount float64 `json:"refunded_amount"`
	Status         string  `json:"status"`
}

// --- Dashboard ---

type AdminDashboardStats struct {
	TotalRevenue       float64                   `json:"total_revenue"`
	ActiveSubscribers  int                       `json:"active_subscribers"`
	TotalUsers         int                       `json:"total_users"`
	ActiveUsers        int                       `json:"active_users"`
	RecentTransactions []TransactionListResponse `json:"recent_transactions"`
}

// --- Plan Management ---

type AdminCreatePlanRequest struct {
	Name          string          `json:"name" validate:"required"`
	Slug          string          `json:"slug" validate:"required"`
	Price         float64         `json:"price" validate:"gte=0"`
	TaxRate       float64         `json:"tax_rate"`
	BillingPeriod string          `json:"billing_period" validate:"required,oneof=monthly yearly"`
	Features      PlanFeaturesDTO `json:"features"` // Use DTO for features JSON
}

type AdminUpdatePlanRequest struct {
	Name          string           `json:"name,omitempty"`
	Description   *string          `json:"description,omitempty"`
	Tagline       *string          `json:"tagline,omitempty"`
	Price         *float64         `json:"price,omitempty"`
	TaxRate       *float64         `json:"tax_rate,omitempty"`
	IsMostPopular *bool            `json:"is_most_popular,omitempty"`
	IsActive      *bool            `json:"is_active,omitempty"`
	SortOrder     *int             `json:"sort_order,omitempty"`
	Features      *PlanFeaturesDTO `json:"features,omitempty"`
}

type PlanFeaturesDTO struct {
	MaxNotebooks             int  `json:"max_notebooks"`
	MaxNotesPerNotebook      int  `json:"max_notes_per_notebook"`
	SemanticSearch           bool `json:"semantic_search"`
	AiChat                   bool `json:"ai_chat"`
	AiChatDailyLimit         int  `json:"ai_chat_daily_limit"`
	SemanticSearchDailyLimit int  `json:"semantic_search_daily_limit"`
}

type AdminPlanResponse struct {
	Id            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	Description   string          `json:"description"`
	Tagline       string          `json:"tagline"`
	Price         float64         `json:"price"`
	TaxRate       float64         `json:"tax_rate"`
	BillingPeriod string          `json:"billing_period"`
	IsMostPopular bool            `json:"is_most_popular"`
	IsActive      bool            `json:"is_active"`
	SortOrder     int             `json:"sort_order"`
	Features      PlanFeaturesDTO `json:"features"`
}

// --- Plan Feature Management (for pricing modal display) ---

type CreatePlanFeatureRequest struct {
	FeatureKey string `json:"feature_key" validate:"required"`
}

// UpdatePlanFeatureRequest removed as link has no properties

type PlanFeatureResponse struct {
	Id          uuid.UUID `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// --- AI Usage Management ---

type UpdateAiLimitRequest struct {
	AiChatDailyUsage         *int `json:"ai_chat_daily_usage" validate:"omitempty,gte=0"`
	SemanticSearchDailyUsage *int `json:"semantic_search_daily_usage" validate:"omitempty,gte=0"`
}

type UpdateAiLimitResponse struct {
	UserId                      uuid.UUID `json:"user_id"`
	PreviousChatUsage           int       `json:"previous_chat_usage"`
	NewChatUsage                int       `json:"new_chat_usage"`
	PreviousSemanticSearchUsage int       `json:"previous_semantic_search_usage"`
	NewSemanticSearchUsage      int       `json:"new_semantic_search_usage"`
	UserEmail                   string    `json:"user_email"`
}

type BulkUpdateAiLimitRequest struct {
	UserIds                  []uuid.UUID `json:"user_ids" validate:"required,min=1"`
	AiChatDailyUsage         *int        `json:"ai_chat_daily_usage" validate:"omitempty,gte=0"`
	SemanticSearchDailyUsage *int        `json:"semantic_search_daily_usage" validate:"omitempty,gte=0"`
}

type BulkResetAiLimitRequest struct {
	UserIds []uuid.UUID `json:"user_ids" validate:"required,min=1"`
}

type BulkAiLimitResponse struct {
	TotalRequested int         `json:"total_requested"`
	TotalUpdated   int         `json:"total_updated"`
	FailedUserIds  []uuid.UUID `json:"failed_user_ids,omitempty"`
}
