package webauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"remnawave-tg-shop-bot/internal/database"
)

const (
	AuthTypeWeb   = "web"
	sessionMaxAge = 60 * 60 * 24 * 30
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	customerRepository *database.CustomerRepository
	sessionSecret      []byte
}

func NewService(customerRepository *database.CustomerRepository, sessionSecret string) *Service {
	return &Service{
		customerRepository: customerRepository,
		sessionSecret:      []byte(sessionSecret),
	}
}

func (s *Service) Register(ctx context.Context, login, password, language string) (*database.Customer, string, error) {
	if login == "" || password == "" {
		return nil, "", fmt.Errorf("login and password are required")
	}

	existingByLogin, err := s.customerRepository.FindByLogin(ctx, login)
	if err != nil {
		return nil, "", err
	}
	if existingByLogin != nil {
		return nil, "", fmt.Errorf("login already registered")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	hashString := string(passwordHash)
	customer, err := s.customerRepository.CreateWebCustomer(ctx, &database.Customer{
		Login:        &login,
		PasswordHash: &hashString,
		Language:     language,
		AuthType:     AuthTypeWeb,
		IsActive:     true,
	})
	if err != nil {
		return nil, "", err
	}

	return customer, s.issueSession(customer.ID), nil
}

func (s *Service) Login(ctx context.Context, identifier, password string) (*database.Customer, string, error) {
	if identifier == "" || password == "" {
		return nil, "", ErrInvalidCredentials
	}

	customer, err := s.customerRepository.FindByLogin(ctx, identifier)
	if err != nil {
		return nil, "", err
	}
	if customer == nil || customer.PasswordHash == nil || !customer.IsActive {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*customer.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	now := time.Now().UTC()
	if err := s.customerRepository.UpdateFields(ctx, customer.ID, map[string]any{
		"last_login_at": now,
	}); err != nil {
		return nil, "", err
	}
	customer.LastLoginAt = &now

	return customer, s.issueSession(customer.ID), nil
}

func (s *Service) ResolveSession(ctx context.Context, token string) (*database.Customer, error) {
	customerID, err := s.parseSession(token)
	if err != nil {
		return nil, err
	}

	customer, err := s.customerRepository.FindById(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if customer == nil || !customer.IsActive {
		if customer != nil && customer.MergedIntoCustomerID != nil {
			mergedCustomer, err := s.customerRepository.FindById(ctx, *customer.MergedIntoCustomerID)
			if err != nil {
				return nil, err
			}
			if mergedCustomer != nil && mergedCustomer.IsActive {
				return mergedCustomer, nil
			}
		}
		return nil, ErrInvalidCredentials
	}

	return customer, nil
}

func (s *Service) issueSession(customerID int64) string {
	expiresAt := time.Now().UTC().Add(sessionMaxAge * time.Second).Unix()
	payload := fmt.Sprintf("%d:%d", customerID, expiresAt)
	signature := s.sign(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + ":" + signature))
}

func (s *Service) parseSession(token string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, ErrInvalidCredentials
	}

	parts := strings.Split(string(raw), ":")
	if len(parts) != 3 {
		return 0, ErrInvalidCredentials
	}

	payload := strings.Join(parts[:2], ":")
	if !hmac.Equal([]byte(parts[2]), []byte(s.sign(payload))) {
		return 0, ErrInvalidCredentials
	}

	customerID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrInvalidCredentials
	}

	expiresAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().UTC().Unix() > expiresAt {
		return 0, ErrInvalidCredentials
	}

	return customerID, nil
}

func (s *Service) sign(payload string) string {
	mac := hmac.New(sha256.New, s.sessionSecret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
