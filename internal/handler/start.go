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
		slog.Error("error finding customer by telegram id", "error", err)
		return
	}

	if existingCustomer == nil {
		existingCustomer, err = h.customerRepository.Create(ctxWithTime, &database.Customer{
			TelegramID: update.Message.Chat.ID,
			Language:   langCode,
		})
		if err != nil {
			slog.Error("error creating customer", "error", err)
			return
		}

		if strings.Contains(update.Message.Text, "ref_") {
			arg := strings.Split(update.Message.Text, " ")[1]
			if strings.HasPrefix(arg, "ref_") {
				code := strings.TrimPrefix(arg, "ref_")
				referrerId, err := strconv.ParseInt(code, 10, 64)
				if err != nil {
					slog.Error("error parsing referrer id", "error", err)
					return
				}
				_, err = h.customerRepository.FindByTelegramId(ctx, referrerId)
				if err == nil {
					_, err := h.referralRepository.Create(ctx, referrerId, existingCustomer.TelegramID)
					if err != nil {
						slog.Error("error creating referral", "error", err)
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
			slog.Error("Error updating customer", "error", err)
			return
		}
	}

	// Handle promo code from deep link (start=promo=CODE)
	if strings.Contains(update.Message.Text, "promo=") {
		parts := strings.Split(update.Message.Text, " ")
		if len(parts) > 1 {
			arg := parts[1]
			if strings.HasPrefix(arg, "promo=") {
				promoCode := strings.TrimPrefix(arg, "promo=")
				h.handlePromoFromStart(ctx, b, update, existingCustomer, promoCode)
				return
			}
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
		slog.Error("Error sending removing reply keyboard", "error", err)
		return
	}

	_, err = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: m.ID,
	})

	if err != nil {
		slog.Error("Error deleting message", "error", err)
		return
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: inlineKeyboard,
		},
		Text: h.translation.GetText(langCode, "greeting"),
	})
	if err != nil {
		slog.Error("Error sending /start message", "error", err)
	}
}

func (h Handler) StartCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	ctxWithTime, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	callback := update.CallbackQuery
	langCode := callback.From.LanguageCode

	existingCustomer, err := h.customerRepository.FindByTelegramId(ctxWithTime, callback.From.ID)
	if err != nil {
		slog.Error("error finding customer by telegram id", "error", err)
		return
	}

	inlineKeyboard := h.buildStartKeyboard(existingCustomer, langCode)

	_, err = b.EditMessageText(ctxWithTime, &bot.EditMessageTextParams{
		ChatID:    callback.Message.Message.Chat.ID,
		MessageID: callback.Message.Message.ID,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: inlineKeyboard,
		},
		Text: h.translation.GetText(langCode, "greeting"),
	})
	if err != nil {
		slog.Error("Error sending /start message", "error", err)
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

	if existingCustomer.SubscriptionLink == nil && !existingCustomer.TrialUsed && config.IsTrialEnabled() {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "trial_button"), CallbackData: CallbackTrial}})
	}

	inlineKeyboard = append(inlineKeyboard, [][]models.InlineKeyboardButton{{{Text: h.translation.GetText(langCode, "buy_button"), CallbackData: CallbackBuy}}}...)

	if existingCustomer.SubscriptionLink != nil && existingCustomer.ExpireAt.After(time.Now()) {
		inlineKeyboard = append(inlineKeyboard, h.resolveConnectButton(langCode))
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{
			{Text: h.translation.GetText(langCode, "my_devices_button"), CallbackData: CallbackDevices},
		})
	}

	if config.GetReferralDays() > 0 {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "referral_button"), CallbackData: CallbackReferral}})
	}

	// Build link buttons based on order from settings
	buttonConfigs := map[string]struct {
		url  string
		text string
	}{
		"server_status": {config.ServerStatusURL(), h.translation.GetText(langCode, "server_status_button")},
		"support":       {config.SupportURL(), h.translation.GetText(langCode, "support_button")},
		"feedback":      {config.FeedbackURL(), h.translation.GetText(langCode, "feedback_button")},
		"channel":       {config.ChannelURL(), h.translation.GetText(langCode, "channel_button")},
	}

	buttonOrder := config.LinkButtonsOrder()
	if len(buttonOrder) == 0 {
		buttonOrder = []string{"server_status", "support", "feedback", "channel"}
	}

	var linkButtons []models.InlineKeyboardButton
	for _, id := range buttonOrder {
		if cfg, ok := buttonConfigs[id]; ok && cfg.url != "" {
			linkButtons = append(linkButtons, models.InlineKeyboardButton{Text: cfg.text, URL: cfg.url})
		}
	}

	// Add link buttons based on layout setting
	if len(linkButtons) > 0 {
		layout := config.LinkButtonsLayout()
		switch layout {
		case "2x2":
			for i := 0; i < len(linkButtons); i += 2 {
				if i+1 < len(linkButtons) {
					inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{linkButtons[i], linkButtons[i+1]})
				} else {
					inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{linkButtons[i]})
				}
			}
		case "1x4":
			inlineKeyboard = append(inlineKeyboard, linkButtons)
		default: // "4x1" or empty - each button in separate row
			for _, btn := range linkButtons {
				inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{btn})
			}
		}
	}

	if config.TosURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "tos_button"), URL: config.TosURL()}})
	}

	// Add admin panel button if user is admin
	if existingCustomer.TelegramID == config.GetAdminTelegramId() && config.BotAdminURL() != "" {
		inlineKeyboard = append(inlineKeyboard, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "admin_panel_button"), WebApp: &models.WebAppInfo{
			URL: config.BotAdminURL(),
		}}})
	}

	return inlineKeyboard
}

func (h Handler) handlePromoFromStart(ctx context.Context, b *bot.Bot, update *models.Update, customer *database.Customer, promoCode string) {
	langCode := customer.Language

	// Validate promo code
	validation, err := h.promo.ValidatePromoCode(ctx, promoCode, customer.ID)
	if err != nil {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   h.translation.GetText(langCode, "error_validating_promo"),
		})
		return
	}

	if !validation.Valid {
		_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   validation.Message,
		})
		return
	}

	// Show promo confirmation
	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{
					Text:         h.translation.GetText(langCode, "apply_promo"),
					CallbackData: fmt.Sprintf("%s_%d", CallbackPromo, validation.PromoID),
				},
			},
			{
				{
					Text:         h.translation.GetText(langCode, "cancel"),
					CallbackData: CallbackStart,
				},
			},
		},
	}

	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        validation.Message,
		ReplyMarkup: keyboard,
	})
}
