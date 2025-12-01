package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/utils"
)

type ConnectHandler struct {
	customerRepository *database.CustomerRepository
	translation        *translation.Manager
	remnawaveClient    *remnawave.Client
}

func NewConnectHandler(
	customerRepository *database.CustomerRepository,
	translation *translation.Manager,
	remnawaveClient *remnawave.Client,
) *ConnectHandler {
	return &ConnectHandler{
		customerRepository: customerRepository,
		translation:        translation,
		remnawaveClient:    remnawaveClient,
	}
}

func (h *ConnectHandler) ConnectCommandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	customer, err := h.customerRepository.FindByTelegramId(ctx, update.Message.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", "error", err)
		return
	}
	if customer == nil {
		slog.Error("customer not exist", "telegramId", utils.MaskHalfInt64(update.Message.Chat.ID), "error", err)
		return
	}

	langCode := update.Message.From.LanguageCode

	isDisabled := true
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      h.buildConnectText(ctx, customer, langCode),
		ParseMode: models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &isDisabled,
		},
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{{Text: h.translation.GetText(langCode, "payment_methods_button"), CallbackData: CallbackPaymentMethods}},
				{{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart}},
			},
		},
	})

	if err != nil {
		slog.Error("Error sending connect message", "error", err)
	}
}

func (h *ConnectHandler) ConnectCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	callback := update.CallbackQuery.Message.Message

	customer, err := h.customerRepository.FindByTelegramId(ctx, callback.Chat.ID)
	if err != nil {
		slog.Error("Error finding customer", "error", err)
		return
	}
	if customer == nil {
		slog.Error("customer not exist", "telegramId", utils.MaskHalfInt64(callback.Chat.ID), "error", err)
		return
	}

	langCode := update.CallbackQuery.From.LanguageCode

	var markup [][]models.InlineKeyboardButton
	if config.IsWepAppLinkEnabled() {
		if customer.SubscriptionLink != nil && customer.ExpireAt.After(time.Now()) {
			markup = append(markup, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "connect_button"),
				WebApp: &models.WebAppInfo{
					URL: *customer.SubscriptionLink,
				}}})
		}
	}
	markup = append(markup, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "payment_methods_button"), CallbackData: CallbackPaymentMethods}})
	markup = append(markup, []models.InlineKeyboardButton{{Text: h.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart}})

	isDisabled := true
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Chat.ID,
		MessageID: callback.ID,
		ParseMode: models.ParseModeHTML,
		Text:      h.buildConnectText(ctx, customer, langCode),
		LinkPreviewOptions: &models.LinkPreviewOptions{
			IsDisabled: &isDisabled,
		},
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: markup,
		},
	})

	if err != nil {
		slog.Error("Error sending connect message", "error", err)
	}
}

func (h *ConnectHandler) buildConnectText(ctx context.Context, customer *database.Customer, langCode string) string {
	var info strings.Builder

	userInfo, err := h.remnawaveClient.GetUserInfo(ctx, customer.TelegramID)
	if err != nil {
		slog.Error("Error getting user info from remnawave", "error", err)
	}

	if userInfo != nil && userInfo.ExpireAt != nil {
		currentTime := time.Now()

		// Status
		var statusText string
		if currentTime.Before(*userInfo.ExpireAt) {
			statusText = h.translation.GetText(langCode, "status_active")
		} else {
			statusText = h.translation.GetText(langCode, "status_expired")
		}
		info.WriteString(fmt.Sprintf("%s\n\n", h.translation.GetText(langCode, "subscription_header")))
		info.WriteString(fmt.Sprintf("%s %s\n", h.translation.GetText(langCode, "status_label"), statusText))

		// Expire date
		formattedDate := userInfo.ExpireAt.Format("2006-01-02")
		info.WriteString(fmt.Sprintf("%s %s\n", h.translation.GetText(langCode, "expire_date_label"), formattedDate))

		// Days left
		if currentTime.Before(*userInfo.ExpireAt) {
			daysLeft := int(userInfo.ExpireAt.Sub(currentTime).Hours() / 24)
			info.WriteString(fmt.Sprintf("%s %d\n", h.translation.GetText(langCode, "days_left_label"), daysLeft))
		} else {
			info.WriteString(fmt.Sprintf("%s 0\n", h.translation.GetText(langCode, "days_left_label")))
		}

		// Subscription link
		if userInfo.SubscriptionURL != "" && !config.IsWepAppLinkEnabled() {
			info.WriteString(fmt.Sprintf("\n%s\n%s\n", h.translation.GetText(langCode, "config_link_label"), userInfo.SubscriptionURL))
		}

		// Traffic info
		info.WriteString(fmt.Sprintf("\n%s\n", h.translation.GetText(langCode, "traffic_header")))
		if userInfo.TrafficLimitBytes == 0 {
			info.WriteString(fmt.Sprintf("%s %s\n", h.translation.GetText(langCode, "traffic_limit_label"), h.translation.GetText(langCode, "unlimited")))
		} else {
			limitGB := float64(userInfo.TrafficLimitBytes) / (1024 * 1024 * 1024)
			info.WriteString(fmt.Sprintf("%s %.2f GB\n", h.translation.GetText(langCode, "traffic_limit_label"), limitGB))
		}

		usedGB := float64(userInfo.UsedTrafficBytes) / (1024 * 1024 * 1024)
		info.WriteString(fmt.Sprintf("%s %.2f GB\n", h.translation.GetText(langCode, "traffic_used_label"), usedGB))
	} else if customer.ExpireAt != nil {
		currentTime := time.Now()

		if currentTime.Before(*customer.ExpireAt) {
			formattedDate := customer.ExpireAt.Format("02.01.2006 15:04")

			subscriptionActiveText := h.translation.GetText(langCode, "subscription_active")
			info.WriteString(fmt.Sprintf(subscriptionActiveText, formattedDate))

			if customer.SubscriptionLink != nil && *customer.SubscriptionLink != "" {
				if !config.IsWepAppLinkEnabled() {
					subscriptionLinkText := h.translation.GetText(langCode, "subscription_link")
					info.WriteString(fmt.Sprintf(subscriptionLinkText, *customer.SubscriptionLink))
				}
			}
		} else {
			noSubscriptionText := h.translation.GetText(langCode, "no_subscription")
			info.WriteString(noSubscriptionText)
		}
	} else {
		noSubscriptionText := h.translation.GetText(langCode, "no_subscription")
		info.WriteString(noSubscriptionText)
	}

	return info.String()
}
