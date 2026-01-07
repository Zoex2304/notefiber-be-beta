package mapper

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"
)

type SubscriptionMapper struct {
	featureMapper *FeatureMapper
}

func NewSubscriptionMapper() *SubscriptionMapper {
	return &SubscriptionMapper{
		featureMapper: NewFeatureMapper(),
	}
}

func (m *SubscriptionMapper) PlanToEntity(p *model.SubscriptionPlan) *entity.SubscriptionPlan {
	if p == nil {
		return nil
	}
	return &entity.SubscriptionPlan{
		Id:                       p.Id,
		Name:                     p.Name,
		Slug:                     p.Slug,
		Description:              p.Description,
		Tagline:                  p.Tagline,
		Price:                    p.Price,
		TaxRate:                  p.TaxRate,
		BillingPeriod:            entity.BillingPeriod(p.BillingPeriod),
		MaxNotebooks:             p.MaxNotebooks,
		MaxNotesPerNotebook:      p.MaxNotesPerNotebook,
		AiChatDailyLimit:         p.AiChatDailyLimit,
		SemanticSearchDailyLimit: p.SemanticSearchDailyLimit,
		SemanticSearchEnabled:    p.SemanticSearchEnabled,
		AiChatEnabled:            p.AiChatEnabled,
		IsMostPopular:            p.IsMostPopular,
		IsActive:                 p.IsActive,
		SortOrder:                p.SortOrder,
		Features:                 m.mapFeaturesToEntities(p.Features),
	}
}

func (m *SubscriptionMapper) PlanToModel(p *entity.SubscriptionPlan) *model.SubscriptionPlan {
	if p == nil {
		return nil
	}
	return &model.SubscriptionPlan{
		Id:                       p.Id,
		Name:                     p.Name,
		Slug:                     p.Slug,
		Description:              p.Description,
		Tagline:                  p.Tagline,
		Price:                    p.Price,
		TaxRate:                  p.TaxRate,
		BillingPeriod:            string(p.BillingPeriod),
		MaxNotebooks:             p.MaxNotebooks,
		MaxNotesPerNotebook:      p.MaxNotesPerNotebook,
		AiChatDailyLimit:         p.AiChatDailyLimit,
		SemanticSearchDailyLimit: p.SemanticSearchDailyLimit,
		SemanticSearchEnabled:    p.SemanticSearchEnabled,
		AiChatEnabled:            p.AiChatEnabled,
		IsMostPopular:            p.IsMostPopular,
		IsActive:                 p.IsActive,
		SortOrder:                p.SortOrder,
		Features:                 m.mapFeaturesToModels(p.Features),
	}
}

func (m *SubscriptionMapper) mapFeaturesToEntities(models []*model.Feature) []entity.Feature {
	if models == nil {
		return nil
	}
	entities := make([]entity.Feature, len(models))
	for i, mdl := range models {
		if val := m.featureMapper.ToEntity(mdl); val != nil {
			entities[i] = *val
		}
	}
	return entities
}

func (m *SubscriptionMapper) mapFeaturesToModels(entities []entity.Feature) []*model.Feature {
	if entities == nil {
		return nil
	}
	models := make([]*model.Feature, len(entities))
	for i, ent := range entities {
		models[i] = m.featureMapper.ToModel(&ent)
	}
	return models
}

func (m *SubscriptionMapper) UserSubscriptionToEntity(s *model.UserSubscription) *entity.UserSubscription {
	if s == nil {
		return nil
	}
	return &entity.UserSubscription{
		Id:                    s.Id,
		UserId:                s.UserId,
		PlanId:                s.PlanId,
		BillingAddressId:      s.BillingAddressId,
		Status:                entity.SubscriptionStatus(s.Status),
		CurrentPeriodStart:    s.CurrentPeriodStart,
		CurrentPeriodEnd:      s.CurrentPeriodEnd,
		PaymentStatus:         entity.PaymentStatus(s.PaymentStatus),
		MidtransTransactionId: s.MidtransTransactionId,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
	}
}

func (m *SubscriptionMapper) UserSubscriptionToModel(s *entity.UserSubscription) *model.UserSubscription {
	if s == nil {
		return nil
	}
	return &model.UserSubscription{
		Id:                    s.Id,
		UserId:                s.UserId,
		PlanId:                s.PlanId,
		BillingAddressId:      s.BillingAddressId,
		Status:                string(s.Status),
		CurrentPeriodStart:    s.CurrentPeriodStart,
		CurrentPeriodEnd:      s.CurrentPeriodEnd,
		PaymentStatus:         string(s.PaymentStatus),
		MidtransTransactionId: s.MidtransTransactionId,
		CreatedAt:             s.CreatedAt,
		UpdatedAt:             s.UpdatedAt,
	}
}
