package handler

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/utils"
)

type MiddlewareHandler struct {
	customerRepository *database.CustomerRepository
	translation        *translation.Manager
}

func NewMiddlewareHandler(
	customerRepository *database.CustomerRepository,
	translation *translation.Manager,
) *MiddlewareHandler {
	return &MiddlewareHandler{
		customerRepository: customerRepository,
		translation:        translation,
	}
}

func (h *MiddlewareHandler) CreateCustomerIfNotExistMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var telegramId int64
		var langCode string
		var username, firstName, lastName *string

		if update.Message != nil {
			telegramId = update.Message.From.ID
			langCode = update.Message.From.LanguageCode
			if update.Message.From.Username != "" {
				username = &update.Message.From.Username
			}
			if update.Message.From.FirstName != "" {
				firstName = &update.Message.From.FirstName
			}
			if update.Message.From.LastName != "" {
				lastName = &update.Message.From.LastName
			}
		} else if update.CallbackQuery != nil {
			telegramId = update.CallbackQuery.From.ID
			langCode = update.CallbackQuery.From.LanguageCode
			if update.CallbackQuery.From.Username != "" {
				username = &update.CallbackQuery.From.Username
			}
			if update.CallbackQuery.From.FirstName != "" {
				firstName = &update.CallbackQuery.From.FirstName
			}
			if update.CallbackQuery.From.LastName != "" {
				lastName = &update.CallbackQuery.From.LastName
			}
		}

		existingCustomer, err := h.customerRepository.FindByTelegramId(ctx, telegramId)
		if err != nil {
			slog.Error("error finding customer by telegram id", "error", err)
			return
		}

		if existingCustomer == nil {
			_, err = h.customerRepository.Create(ctx, &database.Customer{
				TelegramID:  telegramId,
				Language:    langCode,
				TgUsername:  username,
				TgFirstName: firstName,
				TgLastName:  lastName,
			})
			if err != nil {
				slog.Error("error creating customer", "error", err)
				return
			}
		} else {
			updates := map[string]interface{}{
				"language": langCode,
			}
			if username != nil {
				updates["tg_username"] = *username
			}
			if firstName != nil {
				updates["tg_first_name"] = *firstName
			}
			if lastName != nil {
				updates["tg_last_name"] = *lastName
			}
			// User is interacting with bot, so they haven't blocked it
			updates["is_blocked_by_user"] = false

			err = h.customerRepository.UpdateFields(ctx, existingCustomer.ID, updates)
			if err != nil {
				slog.Error("Error updating customer", "error", err)
				return
			}
		}

		next(ctx, b, update)
	}
}

func (h *MiddlewareHandler) SuspiciousUserFilterMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var username, firstName, lastName *string
		var userID int64
		var chatID int64
		var langCode string

		if update.Message != nil {
			username = &update.Message.From.Username
			firstName = &update.Message.From.FirstName
			lastName = &update.Message.From.LastName
			userID = update.Message.From.ID
			chatID = update.Message.Chat.ID
			langCode = update.Message.From.LanguageCode
		} else if update.CallbackQuery != nil {
			username = &update.CallbackQuery.From.Username
			firstName = &update.CallbackQuery.From.FirstName
			lastName = &update.CallbackQuery.From.LastName
			userID = update.CallbackQuery.From.ID
			chatID = update.CallbackQuery.Message.Message.Chat.ID
			langCode = update.CallbackQuery.From.LanguageCode
		} else {
			next(ctx, b, update)
			return
		}

		customer, err := h.customerRepository.FindByTelegramId(ctx, userID)
		if err != nil {
			slog.Error("Failed to check if user is blocked", "error", err)
		} else if customer != nil && customer.IsBlocked {
			slog.Warn("blocked user by is_blocked flag", "userId", utils.MaskHalfInt64(userID))
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      h.translation.GetText(langCode, "access_denied"),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				slog.Error("error sending blocked user message", "error", err)
			}
			return
		}

		if config.GetBlockedTelegramIds()[userID] {
			slog.Warn("blocked user by telegram id", "userId", utils.MaskHalfInt64(userID))
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      h.translation.GetText(langCode, "access_denied"),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				slog.Error("error sending blocked user message", "error", err)
			}
			return
		}

		if config.GetWhitelistedTelegramIds()[userID] {
			slog.Info("whitelisted user allowed", "userId", utils.MaskHalfInt64(userID))
			next(ctx, b, update)
			return
		}

		if utils.IsSuspiciousUser(username, firstName, lastName) {
			slog.Warn("suspicious user blocked", "userId", utils.MaskHalfInt64(userID))
			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      h.translation.GetText(langCode, "access_denied"),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				slog.Error("error sending suspicious user message", "error", err)
			}
			return
		}

		next(ctx, b, update)
	}
}
