// FILE: internal/mapper/feature_mapper.go
// Mapper for Feature entity <-> model conversion
package mapper

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"
)

type FeatureMapper struct{}

func NewFeatureMapper() *FeatureMapper {
	return &FeatureMapper{}
}

func (m *FeatureMapper) ToEntity(model *model.Feature) *entity.Feature {
	if model == nil {
		return nil
	}
	return &entity.Feature{
		Id:          model.Id,
		Key:         model.Key,
		Name:        model.Name,
		Description: model.Description,
		Category:    model.Category,
		IsActive:    model.IsActive,
		SortOrder:   model.SortOrder,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}

func (m *FeatureMapper) ToModel(entity *entity.Feature) *model.Feature {
	if entity == nil {
		return nil
	}
	return &model.Feature{
		Id:          entity.Id,
		Key:         entity.Key,
		Name:        entity.Name,
		Description: entity.Description,
		Category:    entity.Category,
		IsActive:    entity.IsActive,
		SortOrder:   entity.SortOrder,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

func (m *FeatureMapper) ToEntities(models []*model.Feature) []*entity.Feature {
	entities := make([]*entity.Feature, 0, len(models))
	for _, mdl := range models {
		entities = append(entities, m.ToEntity(mdl))
	}
	return entities
}
