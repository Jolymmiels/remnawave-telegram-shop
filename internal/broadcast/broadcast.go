package broadcast

import (
	"context"
	"log/slog"
	"remnawave-tg-shop-bot/internal/database"
	"strings"
	"time"

	"github.com/go-telegram/bot"
)

const (
	// Telegram limits: ~30 messages/second, safe margin
	messageSendDelay = 35 * time.Millisecond
	// Max retries for rate limit errors
	maxRetries = 3
	// Base delay for exponential backoff
	baseRetryDelay = 1 * time.Second
)

type BroadcastStats struct {
	Total   int `json:"total"`
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
	Blocked int `json:"blocked"`
}

type Service struct {
	repo      *database.BroadcastRepository
	customers *database.CustomerRepository
	tgBotApi  *bot.Bot
}

func NewService(repo *database.BroadcastRepository, tgBotApi *bot.Bot, customers *database.CustomerRepository) *Service {
	return &Service{repo: repo, tgBotApi: tgBotApi, customers: customers}
}

func (s *Service) CreateBroadcast(ctx context.Context, content, broadcastType, language string) (*database.Broadcast, error) {
	br := &database.Broadcast{
		Content:  content,
		Type:     broadcastType,
		Language: language,
	}

	created, err := s.repo.CreateBroadcast(ctx, br)
	if err != nil {
		slog.Error("failed to create broadcast", "error", err)
		return nil, err
	}

	var customers *[]database.Customer
	switch broadcastType {
	case database.BroadcastAll:
		customers, err = s.customers.FindAll(ctx)
	case database.BroadcastActive:
		customers, err = s.customers.FindNonExpired(ctx)
	case database.BroadcastInactive:
		customers, err = s.customers.FindExpired(ctx)
	}
	if err != nil {
		slog.Error("failed to find customers", "error", err)
		return nil, err
	}

	if customers == nil || len(*customers) == 0 {
		return created, nil
	}

	stats := s.sendBroadcast(ctx, *customers, content)
	slog.Info("broadcast completed",
		"total", stats.Total,
		"sent", stats.Sent,
		"failed", stats.Failed,
		"blocked", stats.Blocked,
	)

	return created, nil
}

func (s *Service) sendBroadcast(ctx context.Context, customers []database.Customer, content string) BroadcastStats {
	stats := BroadcastStats{Total: len(customers)}

	for _, customer := range customers {
		select {
		case <-ctx.Done():
			slog.Warn("broadcast cancelled", "sent", stats.Sent, "remaining", stats.Total-stats.Sent-stats.Failed-stats.Blocked)
			return stats
		default:
		}

		err := s.sendMessageWithRetry(ctx, customer.TelegramID, content)
		if err != nil {
			if isBlockedError(err) {
				stats.Blocked++
				slog.Debug("user blocked the bot", "telegram_id", customer.TelegramID)
			} else {
				stats.Failed++
				slog.Warn("failed to send message", "telegram_id", customer.TelegramID, "error", err)
			}
		} else {
			stats.Sent++
		}

		// Rate limiting delay between messages
		time.Sleep(messageSendDelay)
	}

	return stats
}

func (s *Service) sendMessageWithRetry(ctx context.Context, chatID int64, text string) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		_, err := s.tgBotApi.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   text,
		})

		if err == nil {
			return nil
		}

		lastErr = err

		// Check if user blocked the bot - don't retry
		if isBlockedError(err) {
			return err
		}

		// Check for rate limit error
		if retryAfter := extractRetryAfter(err); retryAfter > 0 {
			slog.Warn("rate limited, waiting",
				"chat_id", chatID,
				"retry_after", retryAfter,
				"attempt", attempt+1,
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(retryAfter) * time.Second):
				continue
			}
		}

		// For other errors, use exponential backoff
		if attempt < maxRetries {
			backoff := baseRetryDelay * time.Duration(1<<attempt)
			slog.Debug("retrying after error",
				"chat_id", chatID,
				"error", err,
				"backoff", backoff,
				"attempt", attempt+1,
			)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
	}

	return lastErr
}

func isBlockedError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	blockedPhrases := []string{
		"Forbidden",
		"bot was blocked",
		"user is deactivated",
		"chat not found",
		"bot was kicked",
		"have no rights to send",
		"PEER_ID_INVALID",
		"bot can't initiate conversation",
		"403",
		"400",
	}

	for _, phrase := range blockedPhrases {
		if strings.Contains(errStr, phrase) {
			return true
		}
	}

	return false
}

func extractRetryAfter(err error) int {
	if err == nil {
		return 0
	}

	errStr := err.Error()
	if strings.Contains(errStr, "Too Many Requests") || strings.Contains(errStr, "retry after") || strings.Contains(errStr, "429") {
		return 5
	}

	return 0
}

func (s *Service) List(ctx context.Context, params database.BroadcastListParams) (*[]database.Broadcast, error) {
	return s.repo.List(ctx, params)
}
