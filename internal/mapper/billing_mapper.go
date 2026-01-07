package mapper

import (
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/model"
)

type BillingMapper struct{}

func NewBillingMapper() *BillingMapper {
	return &BillingMapper{}
}

func (m *BillingMapper) ToEntity(b *model.BillingAddress) *entity.BillingAddress {
	if b == nil {
		return nil
	}
	return &entity.BillingAddress{
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
	}
}

func (m *BillingMapper) ToModel(b *entity.BillingAddress) *model.BillingAddress {
	if b == nil {
		return nil
	}
	return &model.BillingAddress{
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
	}
}
