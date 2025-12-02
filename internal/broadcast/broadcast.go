package broadcast

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	// Telegram limits: ~30 messages/second, safe margin
	messageSendDelay = 35 * time.Millisecond
	// Max retries for rate limit errors
	maxRetries = 3
	// Base delay for exponential backoff
	baseRetryDelay = 1 * time.Second
	// Log progress every N messages
	progressLogInterval = 100
	// Update DB stats every N messages
	dbUpdateInterval = 50
)

type BroadcastStats struct {
	Total   int `json:"total"`
	Sent    int `json:"sent"`
	Failed  int `json:"failed"`
	Blocked int `json:"blocked"`
}

type MediaInfo struct {
	Data []byte
	Type string // "image" or "video"
}

type Service struct {
	repo      *database.BroadcastRepository
	customers *database.CustomerRepository
	tgBotApi  *bot.Bot
	appCtx    context.Context
}

func NewService(repo *database.BroadcastRepository, tgBotApi *bot.Bot, customers *database.CustomerRepository) *Service {
	return &Service{repo: repo, tgBotApi: tgBotApi, customers: customers}
}

// SetAppContext sets the application context for graceful shutdown support
func (s *Service) SetAppContext(ctx context.Context) {
	s.appCtx = ctx
}

func (s *Service) CreateBroadcastWithMedia(ctx context.Context, content, broadcastType, language string, mediaData []byte, mediaType string) (*database.Broadcast, error) {
	var media *MediaInfo
	if len(mediaData) > 0 {
		mediaKind := "image"
		if strings.HasPrefix(mediaType, "video/") {
			mediaKind = "video"
		}
		media = &MediaInfo{Data: mediaData, Type: mediaKind}
	}
	return s.createBroadcastInternal(ctx, content, broadcastType, language, media)
}

func (s *Service) CreateBroadcast(ctx context.Context, content, broadcastType, language string) (*database.Broadcast, error) {
	return s.createBroadcastInternal(ctx, content, broadcastType, language, nil)
}

func (s *Service) createBroadcastInternal(ctx context.Context, content, broadcastType, language string, media *MediaInfo) (*database.Broadcast, error) {
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
		customers, err = s.customers.FindAllWithLanguage(ctx, language)
	case database.BroadcastActive:
		customers, err = s.customers.FindNonExpiredWithLanguage(ctx, language)
	case database.BroadcastInactive:
		customers, err = s.customers.FindExpiredWithLanguage(ctx, language)
	case database.BroadcastNoSubscription:
		customers, err = s.customers.FindNoSubscriptionWithLanguage(ctx, language)
	}
	if err != nil {
		slog.Error("failed to find customers", "error", err)
		return nil, err
	}

	if customers == nil || len(*customers) == 0 {
		// Mark as completed with zero recipients
		_ = s.repo.UpdateBroadcastStats(ctx, created.ID, database.BroadcastStatusCompleted, 0, 0, 0, 0)
		created.Status = database.BroadcastStatusCompleted
		return created, nil
	}

	// Update total count and set status to in_progress
	totalCount := len(*customers)
	_ = s.repo.UpdateBroadcastStats(ctx, created.ID, database.BroadcastStatusInProgress, totalCount, 0, 0, 0)
	created.Status = database.BroadcastStatusInProgress
	created.TotalCount = totalCount

	// Run broadcast in background with app context for graceful shutdown
	go func() {
		broadcastCtx := context.Background()
		if s.appCtx != nil {
			broadcastCtx = s.appCtx
		}
		s.sendBroadcast(broadcastCtx, created.ID, *customers, content, media)
	}()

	return created, nil
}

func (s *Service) sendBroadcast(ctx context.Context, broadcastID int64, customers []database.Customer, content string, media *MediaInfo) {
	stats := BroadcastStats{Total: len(customers)}
	processed := 0
	var blockedIDs []int64

	slog.Info("broadcast started", "broadcast_id", broadcastID, "total", stats.Total, "has_media", media != nil)

	for _, customer := range customers {
		select {
		case <-ctx.Done():
			slog.Warn("broadcast cancelled due to shutdown", "broadcast_id", broadcastID, "sent", stats.Sent, "remaining", stats.Total-processed)
			if len(blockedIDs) > 0 {
				_ = s.customers.SetBlockedByUserBatch(context.Background(), blockedIDs, true)
			}
			_ = s.repo.UpdateBroadcastStats(context.Background(), broadcastID, database.BroadcastStatusFailed, stats.Total, stats.Sent, stats.Failed, stats.Blocked)
			s.notifyAdmin(context.Background(), broadcastID, stats)
			return
		default:
		}

		var err error
		if media != nil {
			err = s.sendMediaWithRetry(ctx, customer.TelegramID, content, media)
		} else {
			err = s.sendMessageWithRetry(ctx, customer.TelegramID, content)
		}
		if err != nil {
			if isBlockedError(err) {
				stats.Blocked++
				blockedIDs = append(blockedIDs, customer.TelegramID)
			} else {
				stats.Failed++
				slog.Warn("failed to send message", "telegram_id", customer.TelegramID, "error", err)
			}
		} else {
			stats.Sent++
			if customer.IsBlockedByUser {
				_ = s.customers.SetBlockedByUser(ctx, customer.TelegramID, false)
			}
		}
		processed++

		if processed%progressLogInterval == 0 {
			slog.Info("broadcast progress",
				"broadcast_id", broadcastID,
				"processed", processed,
				"total", stats.Total,
				"sent", stats.Sent,
				"failed", stats.Failed,
				"blocked", stats.Blocked,
				"percent", fmt.Sprintf("%.1f%%", float64(processed)/float64(stats.Total)*100),
			)
		}

		if processed%dbUpdateInterval == 0 {
			_ = s.repo.UpdateBroadcastStats(ctx, broadcastID, database.BroadcastStatusInProgress, stats.Total, stats.Sent, stats.Failed, stats.Blocked)
			if len(blockedIDs) > 0 {
				_ = s.customers.SetBlockedByUserBatch(ctx, blockedIDs, true)
				blockedIDs = nil
			}
		}

		time.Sleep(messageSendDelay)
	}

	if len(blockedIDs) > 0 {
		_ = s.customers.SetBlockedByUserBatch(ctx, blockedIDs, true)
	}

	_ = s.repo.UpdateBroadcastStats(ctx, broadcastID, database.BroadcastStatusCompleted, stats.Total, stats.Sent, stats.Failed, stats.Blocked)

	slog.Info("broadcast completed",
		"broadcast_id", broadcastID,
		"total", stats.Total,
		"sent", stats.Sent,
		"failed", stats.Failed,
		"blocked", stats.Blocked,
	)

	s.notifyAdmin(ctx, broadcastID, stats)
}

func (s *Service) notifyAdmin(ctx context.Context, broadcastID int64, stats BroadcastStats) {
	adminID := config.GetAdminTelegramId()
	if adminID == 0 {
		return
	}

	message := fmt.Sprintf(
		"📢 Broadcast #%d completed\n\n"+
			"📊 Statistics:\n"+
			"• Total: %d\n"+
			"• Sent: %d ✅\n"+
			"• Failed: %d ❌\n"+
			"• Blocked: %d 🚫\n\n"+
			"Success rate: %.1f%%",
		broadcastID,
		stats.Total,
		stats.Sent,
		stats.Failed,
		stats.Blocked,
		float64(stats.Sent)/float64(stats.Total)*100,
	)

	_, err := s.tgBotApi.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: adminID,
		Text:   message,
	})
	if err != nil {
		slog.Error("failed to notify admin about broadcast completion", "error", err)
	}
}

func (s *Service) sendMessageWithRetry(ctx context.Context, chatID int64, text string) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		_, err := s.tgBotApi.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      text,
			ParseMode: models.ParseModeHTML,
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

func (s *Service) sendMediaWithRetry(ctx context.Context, chatID int64, caption string, media *MediaInfo) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		var err error

		if media.Type == "video" {
			_, err = s.tgBotApi.SendVideo(ctx, &bot.SendVideoParams{
				ChatID:    chatID,
				Video:     &models.InputFileUpload{Filename: "video.mp4", Data: bytes.NewReader(media.Data)},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})
		} else {
			_, err = s.tgBotApi.SendPhoto(ctx, &bot.SendPhotoParams{
				ChatID:    chatID,
				Photo:     &models.InputFileUpload{Filename: "photo.jpg", Data: bytes.NewReader(media.Data)},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})
		}

		if err == nil {
			return nil
		}

		lastErr = err

		if isBlockedError(err) {
			return err
		}

		if retryAfter := extractRetryAfter(err); retryAfter > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(retryAfter) * time.Second):
				continue
			}
		}

		if attempt < maxRetries {
			backoff := baseRetryDelay * time.Duration(1<<attempt)
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

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
