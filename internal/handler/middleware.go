package handler

import (
	"context"

	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/utils"
)

func (h Handler) CreateCustomerIfNotExistMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		var telegramId int64
		var langCode string
		if update.Message != nil {
			telegramId = update.Message.From.ID
			langCode = update.Message.From.LanguageCode
		} else if update.CallbackQuery != nil {
			telegramId = update.CallbackQuery.From.ID
			langCode = update.CallbackQuery.From.LanguageCode
		}
		existingCustomer, err := h.customerRepository.FindByTelegramId(ctx, telegramId)
		if err != nil {
			slog.Error("error finding customer by telegram id", "error", err)
			return
		}

		if existingCustomer == nil {
			_, err = h.customerRepository.Create(ctx, &database.Customer{
				TelegramID: telegramId,
				Language:   langCode,
			})
			if err != nil {
				slog.Error("error creating customer", "error", err)
				return
			}
		} else {
			updates := map[string]interface{}{
				"language": langCode,
			}

			err = h.customerRepository.UpdateFields(ctx, existingCustomer.ID, updates)
			if err != nil {
				slog.Error("Error updating customer", "error", err)
				return
			}
		}

		next(ctx, b, update)
	}
}

func (h Handler) SuspiciousUserFilterMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
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

func (h Handler) CheckChannelSubscriptionMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if !config.IsChannelSubscriptionRequired() {
			next(ctx, b, update)
			return
		}

		var userID int64
		var chatID int64
		var langCode string

		if update.Message != nil {
			userID = update.Message.From.ID
			chatID = update.Message.Chat.ID
			langCode = update.Message.From.LanguageCode
		} else if update.CallbackQuery != nil {
			userID = update.CallbackQuery.From.ID
			chatID = update.CallbackQuery.Message.Message.Chat.ID
			langCode = update.CallbackQuery.From.LanguageCode
		} else {
			next(ctx, b, update)
			return
		}

		member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
			ChatID: config.RequiredChannelID(),
			UserID: userID,
		})
		if err != nil {
			slog.Error("error checking channel subscription", "error", err)
			next(ctx, b, update)
			return
		}

		if member.Type == models.ChatMemberTypeMember ||
			member.Type == models.ChatMemberTypeOwner ||
			member.Type == models.ChatMemberTypeAdministrator ||
			member.Type == models.ChatMemberTypeRestricted {
			next(ctx, b, update)
			return
		}

		kb := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text: h.translation.GetText(langCode, "subscribe_button"),
						URL:  config.ChannelURL(),
					},
				},
				{
					{
						Text:         h.translation.GetText(langCode, "check_subscription_button"),
						CallbackData: CallbackCheckSubscription,
					},
				},
			},
		}

		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      chatID,
			Text:        h.translation.GetText(langCode, "not_subscribed"),
			ParseMode:   models.ParseModeHTML,
			ReplyMarkup: kb,
		})
		if err != nil {
			slog.Error("error sending not subscribed message", "error", err)
		}
	}
}

func (h Handler) AnswerCallbackQueryMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		next(ctx, b, update)
		if update.CallbackQuery != nil {
			_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
			})

			if err != nil {
				slog.Error("error answering callback query", "error", err)
			}
		}
	}
}
