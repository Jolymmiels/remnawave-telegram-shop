// Package menu keeps navigation on one message, with an optional cached photo.
package menu

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// PhotoRecord is scoped to both the bot and the configured file path.
type PhotoRecord struct {
	Hash   string
	FileID string
}

type Store interface {
	LoadPhoto(context.Context, int64, string) (PhotoRecord, error)
	SavePhoto(context.Context, int64, string, PhotoRecord) error
	ListPins(context.Context, int64, int64) ([]int, error)
	TrackPin(context.Context, int64, int64, int) error
	ForgetPin(context.Context, int64, int64, int) error
}

type Service struct {
	store      Store
	botID      int64
	path       string
	data       []byte
	digest     string
	photoMu    sync.Mutex
	fileID     string
	photoDirty bool
	// Bounded locks serialize concurrent /start requests in the same chat.
	chatLocks [64]sync.Mutex
}

// New loads the source once. Replacing the file takes effect after a restart.
// If the file is temporarily absent, a persisted Telegram file_id still works.
func New(ctx context.Context, store Store, botID int64, path string) (*Service, error) {
	s := &Service{store: store, botID: botID, path: path}
	if path == "" {
		return s, nil
	}
	s.path = filepath.Clean(path)
	record, err := store.LoadPhoto(ctx, botID, s.path)
	if err != nil {
		return nil, fmt.Errorf("load menu photo cache: %w", err)
	}
	f, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) && record.FileID != "" {
		s.fileID, s.digest = record.FileID, record.Hash
		slog.Warn("Menu photo source missing; using cached Telegram file", "path", s.path)
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open menu photo: %w", err)
	}
	defer f.Close()
	const maxPhotoSize = 10 * 1024 * 1024
	s.data, err = io.ReadAll(io.LimitReader(f, maxPhotoSize+1))
	if err != nil {
		return nil, fmt.Errorf("read menu photo: %w", err)
	}
	if len(s.data) > maxPhotoSize {
		return nil, errors.New("menu photo must be at most 10 MB")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(s.data))
	if err != nil {
		return nil, fmt.Errorf("menu photo must be a valid JPEG or PNG: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width+cfg.Height > 10000 ||
		float64(cfg.Width)/float64(cfg.Height) > 20 || float64(cfg.Height)/float64(cfg.Width) > 20 {
		return nil, errors.New("menu photo dimensions exceed Telegram limits (width + height <= 10000, aspect ratio <= 20)")
	}
	s.digest = fmt.Sprintf("%x", sha256.Sum256(s.data))
	if record.Hash == s.digest {
		s.fileID = record.FileID
	}
	return s, nil
}

func (s *Service) Send(ctx context.Context, b *bot.Bot, chatID int64, text string, keyboard models.ReplyMarkup) (*models.Message, error) {
	if s.path == "" {
		disabled := true
		return b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID, Text: text, ParseMode: models.ParseModeHTML, ReplyMarkup: keyboard,
			LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: &disabled},
		})
	}
	s.photoMu.Lock()
	if s.photoDirty {
		s.persistPhoto(ctx)
	}
	fileID := s.fileID
	if fileID == "" {
		defer s.photoMu.Unlock()
		return s.upload(ctx, b, chatID, text, keyboard)
	}
	s.photoMu.Unlock()
	message, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID: chatID, Photo: &models.InputFileString{Data: fileID},
		Caption: text, ParseMode: models.ParseModeHTML, ReplyMarkup: keyboard,
	})
	if !invalidFileID(err) {
		return message, err
	}
	// Retry only a rejected file_id, never timeouts or arbitrary 400 errors:
	// those may have delivered the message or indicate an invalid caption.
	s.photoMu.Lock()
	defer s.photoMu.Unlock()
	if s.fileID != fileID && s.fileID != "" {
		return b.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID: chatID, Photo: &models.InputFileString{Data: s.fileID},
			Caption: text, ParseMode: models.ParseModeHTML, ReplyMarkup: keyboard,
		})
	}
	return s.upload(ctx, b, chatID, text, keyboard)
}

// upload is called with photoMu held so the first requests upload only once.
func (s *Service) upload(ctx context.Context, b *bot.Bot, chatID int64, text string, keyboard models.ReplyMarkup) (*models.Message, error) {
	if len(s.data) == 0 {
		return nil, errors.New("cached menu photo was rejected; restore the source file and restart the bot")
	}
	message, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID: chatID, Photo: &models.InputFileUpload{Filename: filepath.Base(s.path), Data: bytes.NewReader(s.data)},
		Caption: text, ParseMode: models.ParseModeHTML, ReplyMarkup: keyboard,
	})
	if err != nil {
		return nil, err
	}
	if len(message.Photo) > 0 {
		s.fileID = message.Photo[len(message.Photo)-1].FileID
		s.photoDirty = true
		s.persistPhoto(ctx)
	}
	return message, nil
}

// persistPhoto is called with photoMu held. Failed writes retry on the next send.
func (s *Service) persistPhoto(ctx context.Context) {
	if err := s.store.SavePhoto(ctx, s.botID, s.path, PhotoRecord{Hash: s.digest, FileID: s.fileID}); err != nil {
		slog.Error("Failed to persist menu photo cache", "error", err)
		return
	}
	s.photoDirty = false
}

func invalidFileID(err error) bool {
	if !errors.Is(err, bot.ErrorBadRequest) {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "wrong file identifier") || strings.Contains(text, "wrong remote file identifier") ||
		strings.Contains(text, "file_id_invalid") || strings.Contains(text, "file reference expired") ||
		strings.Contains(text, "file_reference_expired")
}

func (s *Service) SendStart(ctx context.Context, b *bot.Bot, chatID int64, text string, keyboard models.ReplyMarkup) (*models.Message, error) {
	lock := &s.chatLocks[uint64(chatID)%uint64(len(s.chatLocks))]
	lock.Lock()
	defer lock.Unlock()
	message, err := s.Send(ctx, b, chatID, text, keyboard)
	if err != nil {
		return nil, err
	}
	if err := s.replacePin(ctx, b, chatID, message.ID); err != nil {
		// Lack of pin permissions must not prevent using the delivered menu.
		slog.Warn("Failed to update pinned start menu", "error", err)
	}
	return message, nil
}

func (s *Service) replacePin(ctx context.Context, b *bot.Bot, chatID int64, messageID int) error {
	oldPins, err := s.store.ListPins(ctx, s.botID, chatID)
	if err != nil {
		return err
	}
	// Persist intent before the API call, so even a crash/timeout cannot leave
	// an untracked bot pin. Failed unpins remain tracked for the next /start.
	if err := s.store.TrackPin(ctx, s.botID, chatID, messageID); err != nil {
		return err
	}
	if _, err := b.PinChatMessage(ctx, &bot.PinChatMessageParams{
		ChatID: chatID, MessageID: messageID, DisableNotification: true,
	}); err != nil {
		return err
	}
	var cleanupErrors []error
	for _, oldID := range oldPins {
		if oldID == messageID || oldID == 0 {
			continue
		}
		_, err := b.UnpinChatMessage(ctx, &bot.UnpinChatMessageParams{ChatID: chatID, MessageID: oldID})
		if err != nil && !missingPin(err) {
			cleanupErrors = append(cleanupErrors, err)
			continue
		}
		if err := s.store.ForgetPin(ctx, s.botID, chatID, oldID); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}

func missingPin(err error) bool {
	if !errors.Is(err, bot.ErrorBadRequest) {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "message to unpin not found") || strings.Contains(text, "message is not pinned") ||
		strings.Contains(text, "message_id_invalid")
}

// Edit preserves the existing photo and message ID (including its pin).
// Older text-only menus continue to work after enabling or disabling photos.
func Edit(ctx context.Context, b *bot.Bot, message *models.Message, params *bot.EditMessageTextParams) (*models.Message, error) {
	if message == nil {
		return nil, errors.New("menu message is inaccessible")
	}
	var result *models.Message
	var err error
	if len(message.Photo) > 0 {
		result, err = b.EditMessageCaption(ctx, &bot.EditMessageCaptionParams{
			ChatID: message.Chat.ID, MessageID: message.ID, Caption: params.Text,
			ParseMode: params.ParseMode, CaptionEntities: params.Entities, ReplyMarkup: params.ReplyMarkup,
		})
	} else {
		result, err = b.EditMessageText(ctx, params)
	}
	if errors.Is(err, bot.ErrorBadRequest) && strings.Contains(strings.ToLower(err.Error()), "message is not modified") {
		return message, nil
	}
	return result, err
}
