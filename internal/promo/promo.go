package promo

import (
	"context"
	"fmt"
	"private-remnawave-telegram-shop-bot/internal/database"
	"strings"
	"time"
)

type Service struct {
	repository *database.PromoRepository
}

func NewService(repository *database.PromoRepository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, req *database.CreatePromoRequest) (*database.Promo, error) {
	// Validate input
	req.Code = strings.TrimSpace(strings.ToUpper(req.Code))
	if req.Code == "" {
		return nil, fmt.Errorf("promo code cannot be empty")
	}
	if req.BonusDays <= 0 {
		return nil, fmt.Errorf("bonus days must be greater than 0")
	}
	if req.MaxUses != nil && *req.MaxUses <= 0 {
		return nil, fmt.Errorf("max uses must be greater than 0")
	}

	// Check if promo code already exists
	existing, err := s.repository.GetByCode(ctx, req.Code)
	if err != nil {
		return nil, fmt.Errorf("error checking existing promo: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("promo code '%s' already exists", req.Code)
	}

	return s.repository.Create(ctx, req)
}

func (s *Service) ValidatePromoCode(ctx context.Context, code string, customerID int64) (*database.ValidatePromoResponse, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	
	promo, err := s.repository.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("error retrieving promo: %w", err)
	}

	// Promo code doesn't exist
	if promo == nil {
		return &database.ValidatePromoResponse{
			Valid:   false,
			Message: "Promo code not found",
		}, nil
	}

	// Check if promo is active
	if !promo.Active {
		return &database.ValidatePromoResponse{
			Valid:   false,
			Message: "Promo code is no longer active",
		}, nil
	}

	// Check if promo has expired
	if promo.ExpiresAt != nil && time.Now().After(*promo.ExpiresAt) {
		return &database.ValidatePromoResponse{
			Valid:   false,
			Message: "Promo code has expired",
		}, nil
	}

	// Check if customer has already used this promo
	hasUsed, err := s.repository.HasCustomerUsedPromo(ctx, promo.ID, customerID)
	if err != nil {
		return nil, fmt.Errorf("error checking promo usage: %w", err)
	}
	if hasUsed {
		return &database.ValidatePromoResponse{
			Valid:   false,
			Message: "You have already used this promo code",
		}, nil
	}

	// Check if max uses reached
	if promo.MaxUses != nil && promo.UsedCount >= *promo.MaxUses {
		return &database.ValidatePromoResponse{
			Valid:   false,
			Message: "Promo code usage limit reached",
		}, nil
	}

	// All validations passed
	return &database.ValidatePromoResponse{
		Valid:     true,
		BonusDays: promo.BonusDays,
		Message:   fmt.Sprintf("Promo code applied! +%d bonus days", promo.BonusDays),
		PromoID:   promo.ID,
	}, nil
}

func (s *Service) ApplyPromoCode(ctx context.Context, promoID, customerID int64) error {
	return s.repository.RecordPromoUsage(ctx, promoID, customerID)
}

func (s *Service) GetAll(ctx context.Context) ([]*database.Promo, error) {
	return s.repository.GetAll(ctx)
}

func (s *Service) Update(ctx context.Context, id int64, active bool) error {
	return s.repository.Update(ctx, id, active)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repository.Delete(ctx, id)
}

func (s *Service) GetPromoByID(ctx context.Context, id int64) (*database.Promo, error) {
	return s.repository.GetByID(ctx, id)
}
