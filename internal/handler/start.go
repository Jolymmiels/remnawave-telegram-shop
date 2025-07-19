package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"log/slog"

	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/utils"
)

func (h Handler) StartCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctxWithTime, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	langCode := update.Message.From.LanguageCode
	existingCustomer, err := h.customerRepository.FindByTelegramId(ctx, update.Message.Chat.ID)
	if err != nil {
		slog.Error("error finding customer by telegram id", err)
		return
	}

	if existingCustomer == nil {
		existingCustomer, err = h.customerRepository.Create(ctxWithTime, &database.Customer{
			TelegramID: update.Message.Chat.ID,
			Language:   langCode,
			Balance:    0,
		})
		if err != nil {
			slog.Error("error creating customer", err)
			return
		}

		if strings.Contains(update.Message.Text, "ref_") {
			arg := strings.Split(update.Message.Text, " ")[1]
			if strings.HasPrefix(arg, "ref_") {
				code := strings.TrimPrefix(arg, "ref_")
				referrerId, err := strconv.ParseInt(code, 10, 64)
				if err != nil {
					slog.Error("error parsing referrer id", err)
					return
				}
				_, err = h.customerRepository.FindByTelegramId(ctx, referrerId)
				if err == nil {
					_, err := h.referralRepository.Create(ctx, referrerId, existingCustomer.TelegramID)
					if err != nil {
						slog.Error("error creating referral", err)
						return
					}
					slog.Info("referral created", "referrerId", utils.MaskHalfInt64(referrerId), "refereeId", utils.MaskHalfInt64(existingCustomer.TelegramID))
				}
			}
		}
	} else {
		updates := map[string]interface{}{
			"language": langCode,
		}

		err = h.customerRepository.UpdateFields(ctx, existingCustomer.ID, updates)
		if err != nil {
			slog.Error("Error updating customer", err)
			return
		}
	}

	inlineKeyboard := h.buildStartKeyboard(existingCustomer, langCode)

	m, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "🧹",
		ReplyMarkup: models.ReplyKeyboardRemove{
			RemoveKeyboard: true,
		},
	})

	if err != nil {
		slog.Error("Error sending removing reply keyboard", err)
		return
	}

	_, err = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: m.ID,
	})

	if err != nil {
		slog.Error("Error deleting message", err)
		return
	}

	text := h.translation.GetText(langCode, "greeting") + "\n\n" + h.buildAccountInfo(ctxWithTime, existingCustomer, langCode)
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: inlineKeyboard,
		},
		Text: text,
	})
	if err != nil {
		slog.Error("Error sending /start message", err)
	}
}

func (h Handler) StartCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctxWithTime, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	callback := update.CallbackQuery
	langCode := callback.From.LanguageCode

	existingCustomer, err := h.customerRepository.FindByTelegramId(ctxWithTime, callback.From.ID)
	if err != nil {
		slog.Error("error finding customer by telegram id", err)
		return
	}

	inlineKeyboard := h.buildStartKeyboard(existingCustomer, langCode)

	text := h.translation.GetText(langCode, "greeting") + "\n\n" + h.buildAccountInfo(ctxWithTime, existingCustomer, langCode)
	_, err = b.EditMessageText(ctxWithTime, &bot.EditMessageTextParams{
		ChatID:    callback.Message.Message.Chat.ID,
		MessageID: callback.Message.Message.ID,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: inlineKeyboard,
		},
		Text: text,
	})
	if err != nil {
		slog.Error("Error sending /start message", err)
	}
}

func (h Handler) resolveConnectButton(lang string) []models.InlineKeyboardButton {
	var inlineKeyboard []models.InlineKeyboardButton

	if config.GetMiniAppURL() != "" {
		inlineKeyboard = []models.InlineKeyboardButton{
			{Text: h.translation.GetText(lang, "connect_button"), WebApp: &models.WebAppInfo{
				URL: config.GetMiniAppURL(),
			}},
		}
	} else {
		inlineKeyboard = []models.InlineKeyboardButton{
			{Text: h.translation.GetText(lang, "connect_button"), CallbackData: CallbackConnect},
		}
	}
	return inlineKeyboard
}

func (h Handler) buildStartKeyboard(existingCustomer *database.Customer, langCode string) [][]models.InlineKeyboardButton {
	var inlineKeyboard [][]models.InlineKeyboardButton

	if existingCustomer.SubscriptionLink == nil && config.TrialDays() > 0 {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "trial_button"), CallbackData: CallbackTrial}})
	}

	inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "balance_menu_button"), CallbackData: CallbackBalance}})

	if existingCustomer.SubscriptionLink != nil && existingCustomer.ExpireAt.After(time.Now()) {
		inlineKeyboard = append(inlineKeyboard, h.resolveConnectButton(langCode))
	}

	if config.GetReferralDays() > 0 {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "referral_button"), CallbackData: CallbackReferral}})
	}

	if config.ServerStatusURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "server_status_button"), URL: config.ServerStatusURL()}})
	}

	if config.SupportURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "support_button"), URL: config.SupportURL()}})
	}

	if config.FeedbackURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "feedback_button"), URL: config.FeedbackURL()}})
	}

	if config.ChannelURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "channel_button"), URL: config.ChannelURL()}})
	}

	if config.TosURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "tos_button"), URL: config.TosURL()}})
	}
	return inlineKeyboard
}

func (h Handler) buildAccountInfo(ctx context.Context, customer *database.Customer, lang string) string {
	user, _ := h.paymentService.GetUser(ctx, customer.TelegramID)
	var info strings.Builder
	if user != nil {
		if user.ExpireAt.After(time.Now()) {
			info.WriteString(h.translation.GetText(lang, "subscription_active_hint"))
		} else {
			info.WriteString(h.translation.GetText(lang, "subscription_inactive_hint"))
		}
		info.WriteString("\n\n")
		expire := user.ExpireAt.Format("02.01.2006 15:04")
		status := "ACTIVE"
		if user.Status.Set {
			status = string(user.Status.Value)
		}
		lastClient := "-"
		if !user.LastConnectedNode.Null {
			lastClient = user.LastConnectedNode.Value.GetNodeName()
		}
		start := time.Now().Truncate(24 * time.Hour)
		usage, _ := h.paymentService.GetUserDailyUsage(ctx, user.UUID.String(), start, time.Now())
		limit := 0.0
		if v, ok := user.TrafficLimitBytes.Get(); ok {
			limit = float64(v)
		}
		info.WriteString("📰 Информация об аккаунте:\n\n")
		info.WriteString(fmt.Sprintf("├ Баланс: <%.0f ₽>\n", customer.Balance))
		info.WriteString(fmt.Sprintf("├ Подписка до: <%s>\n", expire))
		info.WriteString(fmt.Sprintf("├ Статус: <%s>\n", status))
		info.WriteString(fmt.Sprintf("├ Последний клиент: <%s>\n\n", lastClient))
		info.WriteString("🌐 Информация о трафике:\n\n")
		info.WriteString(fmt.Sprintf("├ Лимит в сутки: <%s / %s>\n", utils.FormatGB(usage), utils.FormatGB(limit)))
		info.WriteString(fmt.Sprintf("├ Всего использовано: <%s>\n", utils.FormatGB(user.LifetimeUsedTrafficBytes)))
		untilReset := start.Add(24 * time.Hour).Sub(time.Now())
		info.WriteString(fmt.Sprintf("├ До сброса трафика: <%s>", untilReset.Truncate(time.Second)))
	} else {
		info.WriteString(fmt.Sprintf(h.translation.GetText(lang, "balance_info"), int(customer.Balance)))
	}
	return info.String()
}
