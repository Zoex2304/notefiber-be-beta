package dashboard

import (
	"context"
	"time"

	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/logger"
	"ai-notetaking-be/internal/repository/unitofwork"
)

// Aggregator handles dashboard statistics
type Aggregator struct {
	logger logger.ILogger
}

// NewAggregator creates a new dashboard aggregator
func NewAggregator(logger logger.ILogger) *Aggregator {
	return &Aggregator{
		logger: logger,
	}
}

// GetStats retrieves dashboard statistics
func (a *Aggregator) GetStats(ctx context.Context, uow unitofwork.UnitOfWork) (*dto.AdminDashboardStats, error) {
	totalUsers, err := uow.UserRepository().Count(ctx)
	if err != nil {
		return nil, err
	}

	activeUsers, err := uow.UserRepository().CountByStatus(ctx, string(entity.UserStatusActive))
	if err != nil {
		return nil, err
	}

	totalRevenue, err := uow.SubscriptionRepository().GetTotalRevenue(ctx)
	if err != nil {
		return nil, err
	}

	activeSubs, err := uow.SubscriptionRepository().CountActiveSubscribers(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch Recent Transactions (Limit 5)
	recentTxs, err := uow.SubscriptionRepository().GetTransactions(ctx, "", 5, 0)
	var recentTxDtos []dto.TransactionListResponse
	if err == nil {
		for _, t := range recentTxs {
			recentTxDtos = append(recentTxDtos, dto.TransactionListResponse{
				Id:              t.Id,
				UserId:          t.UserId,
				UserEmail:       t.UserEmail,
				PlanName:        t.PlanName,
				Amount:          t.Amount,
				Status:          string(t.Status),
				PaymentStatus:   string(t.PaymentStatus),
				TransactionDate: t.CreatedAt,
				MidtransOrderId: t.MidtransOrderId,
			})
		}
	}

	return &dto.AdminDashboardStats{
		TotalUsers:         int(totalUsers),
		ActiveUsers:        int(activeUsers),
		TotalRevenue:       totalRevenue,
		ActiveSubscribers:  activeSubs,
		RecentTransactions: recentTxDtos,
	}, nil
}

// GetUserGrowth retrieves user growth statistics
func (a *Aggregator) GetUserGrowth(ctx context.Context, uow unitofwork.UnitOfWork) ([]*dto.UserGrowthStats, error) {
	stats, err := uow.UserRepository().GetUserGrowth(ctx)
	if err != nil {
		return nil, err
	}
	var res []*dto.UserGrowthStats
	for _, st := range stats {
		dateStr, _ := st["date"].(string)
		countVal, _ := st["count"].(int64)

		res = append(res, &dto.UserGrowthStats{
			Date:  dateStr,
			Count: int(countVal),
		})
	}
	return res, nil
}

// GetTransactions retrieves paginated transactions
func (a *Aggregator) GetTransactions(ctx context.Context, uow unitofwork.UnitOfWork, page, limit int, status string) ([]*dto.TransactionListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	txs, err := uow.SubscriptionRepository().GetTransactions(ctx, status, limit, offset)
	if err != nil {
		return nil, err
	}

	var res []*dto.TransactionListResponse
	for _, t := range txs {
		res = append(res, &dto.TransactionListResponse{
			Id:              t.Id,
			UserId:          t.UserId,
			UserEmail:       t.UserEmail,
			PlanName:        t.PlanName,
			Amount:          t.Amount,
			Status:          string(t.Status),
			PaymentStatus:   string(t.PaymentStatus),
			TransactionDate: t.CreatedAt,
			MidtransOrderId: t.MidtransOrderId,
		})
	}
	return res, nil
}

// GetSystemLogs retrieves system logs
func (a *Aggregator) GetSystemLogs(ctx context.Context, loggerSvc logger.ILogger, page, limit int, level string) ([]*dto.LogListResponse, error) {
	logs, err := loggerSvc.GetLogs(level, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}

	var res []*dto.LogListResponse
	for _, l := range logs {
		ts, _ := time.Parse(time.RFC3339, l.Timestamp)
		res = append(res, &dto.LogListResponse{
			Id:        l.Id,
			Level:     l.Level,
			Module:    l.Module,
			Message:   l.Message,
			CreatedAt: ts,
		})
	}
	return res, nil
}

// GetLogDetail retrieves a single log entry
func (a *Aggregator) GetLogDetail(ctx context.Context, loggerSvc logger.ILogger, logId string) (*dto.LogDetailResponse, error) {
	l, err := loggerSvc.GetLogById(logId)
	if err != nil {
		return nil, err
	}

	ts, _ := time.Parse(time.RFC3339, l.Timestamp)
	detailsMap := l.Details

	return &dto.LogDetailResponse{
		LogListResponse: dto.LogListResponse{
			Id:        logId,
			Level:     l.Level,
			Module:    l.Module,
			Message:   l.Message,
			CreatedAt: ts,
		},
		Details: detailsMap,
	}, nil
}
