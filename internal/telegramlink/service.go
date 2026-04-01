package telegramlink

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
)

const tokenTTL = 10 * time.Minute

var ErrBotURLNotConfigured = errors.New("telegram bot url is not configured")

type Service struct {
	customerRepository     *database.CustomerRepository
	telegramLinkRepository *database.TelegramLinkRepository
}

type CreateLinkResult struct {
	URL       string
	ExpiresAt time.Time
}

func NewService(
	customerRepository *database.CustomerRepository,
	telegramLinkRepository *database.TelegramLinkRepository,
) *Service {
	return &Service{
		customerRepository:     customerRepository,
		telegramLinkRepository: telegramLinkRepository,
	}
}

func (s *Service) CreateLink(ctx context.Context, customerID int64) (*CreateLinkResult, error) {
	customer, err := s.customerRepository.FindById(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil {
		return nil, database.ErrTelegramLinkCustomerAbsent
	}
	if customer.TelegramID != 0 {
		return nil, database.ErrCustomerAlreadyLinked
	}

	botURL := config.BotURL()
	if botURL == "" {
		return nil, ErrBotURLNotConfigured
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().UTC().Add(tokenTTL)
	if _, err := s.telegramLinkRepository.Create(ctx, customerID, token, expiresAt); err != nil {
		return nil, err
	}

	return &CreateLinkResult{
		URL:       fmt.Sprintf("%s?start=link_%s", botURL, token),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) ConsumeLink(ctx context.Context, token string, telegramID int64, language string) (*database.Customer, error) {
	return s.telegramLinkRepository.Consume(ctx, token, telegramID, language)
}

func generateToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate telegram link token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}
