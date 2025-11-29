package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/utils"
)

type DevicesHandler struct {
	remnawaveClient    *remnawave.Client
	customerRepository *database.CustomerRepository
	purchaseRepository *database.PurchaseRepository
	planRepository     *database.PlanRepository
	translation        interface {
		GetText(lang string, key string) string
	}
}

func NewDevicesHandler(
	remnawaveClient *remnawave.Client,
	customerRepository *database.CustomerRepository,
	purchaseRepository *database.PurchaseRepository,
	planRepository *database.PlanRepository,
	translation interface {
		GetText(lang string, key string) string
	},
) *DevicesHandler {
	return &DevicesHandler{
		remnawaveClient:    remnawaveClient,
		customerRepository: customerRepository,
		purchaseRepository: purchaseRepository,
		planRepository:     planRepository,
		translation:        translation,
	}
}

func maskHwid(hwid string) string {
	if len(hwid) <= 8 {
		return hwid
	}
	return hwid[:4] + "***" + hwid[len(hwid)-4:]
}

func (dh *DevicesHandler) DevicesCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	callback := update.CallbackQuery
	langCode := callback.From.LanguageCode
	telegramId := callback.From.ID

	userUuid, err := dh.remnawaveClient.GetUserUuidByTelegramId(ctx, telegramId)
	if err != nil {
		slog.Error("Failed to get user UUID", "error", err, "telegramId", utils.MaskHalfInt64(telegramId))
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    callback.Message.Message.Chat.ID,
			MessageID: callback.Message.Message.ID,
			Text:      dh.translation.GetText(langCode, "devices_error"),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: dh.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart}},
				},
			},
		})
		return
	}

	devices, err := dh.remnawaveClient.GetUserDevices(ctx, userUuid)
	if err != nil {
		slog.Error("Failed to get user devices", "error", err)
		_, _ = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    callback.Message.Message.Chat.ID,
			MessageID: callback.Message.Message.ID,
			Text:      dh.translation.GetText(langCode, "devices_error"),
			ReplyMarkup: models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{{Text: dh.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart}},
				},
			},
		})
		return
	}

	// Get device limit from customer's plan or trial settings
	var deviceLimit *int
	customer, err := dh.customerRepository.FindByTelegramId(ctx, telegramId)
	if err == nil && customer != nil {
		lastPurchase, _ := dh.purchaseRepository.FindLastPaidPurchaseWithPlan(ctx, customer.ID)
		if lastPurchase != nil && lastPurchase.PlanID != nil {
			plan, _ := dh.planRepository.FindById(ctx, *lastPurchase.PlanID)
			if plan != nil {
				deviceLimit = plan.DeviceLimit
			}
		} else if customer.TrialUsed {
			// Use trial device limit if user is on trial
			trialLimit := config.TrialDeviceLimit()
			if trialLimit > 0 {
				deviceLimit = &trialLimit
			}
		}
	}

	var text strings.Builder
	text.WriteString(dh.translation.GetText(langCode, "devices_title"))
	if deviceLimit != nil {
		text.WriteString(fmt.Sprintf(" (%d/%d)", len(devices), *deviceLimit))
	}
	text.WriteString("\n\n")

	var keyboard [][]models.InlineKeyboardButton

	if len(devices) == 0 {
		text.WriteString(dh.translation.GetText(langCode, "devices_empty"))
	} else {
		for i, device := range devices {
			deviceName := device.DeviceModel
			if deviceName == "" {
				deviceName = device.Platform
			}
			if deviceName == "" {
				deviceName = "Unknown"
			}

			maskedHwid := maskHwid(device.Hwid)
			text.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, deviceName, maskedHwid))

			keyboard = append(keyboard, []models.InlineKeyboardButton{
				{
					Text:         fmt.Sprintf("🗑 %s (%s)", deviceName, maskedHwid),
					CallbackData: fmt.Sprintf("%s%s", CallbackDeviceDelete, device.Hwid),
				},
			})
		}
	}

	keyboard = append(keyboard, []models.InlineKeyboardButton{
		{Text: dh.translation.GetText(langCode, "back_button"), CallbackData: CallbackStart},
	})

	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Message.Message.Chat.ID,
		MessageID: callback.Message.Message.ID,
		Text:      text.String(),
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	if err != nil {
		slog.Error("Failed to send devices message", "error", err)
	}
}

func (dh *DevicesHandler) DeviceDeleteCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	callback := update.CallbackQuery
	langCode := callback.From.LanguageCode

	hwid := strings.TrimPrefix(callback.Data, CallbackDeviceDelete)
	if hwid == "" {
		return
	}

	maskedHwid := maskHwid(hwid)
	text := fmt.Sprintf(dh.translation.GetText(langCode, "device_delete_confirm"), maskedHwid)

	keyboard := [][]models.InlineKeyboardButton{
		{
			{Text: "✅ " + dh.translation.GetText(langCode, "yes"), CallbackData: fmt.Sprintf("%s%s", CallbackDeviceConfirm, hwid)},
			{Text: "❌ " + dh.translation.GetText(langCode, "no"), CallbackData: CallbackDevices},
		},
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    callback.Message.Message.Chat.ID,
		MessageID: callback.Message.Message.ID,
		Text:      text,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	if err != nil {
		slog.Error("Failed to send delete confirmation", "error", err)
	}
}

func (dh *DevicesHandler) DeviceConfirmDeleteHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	callback := update.CallbackQuery
	langCode := callback.From.LanguageCode
	telegramId := callback.From.ID

	hwid := strings.TrimPrefix(callback.Data, CallbackDeviceConfirm)
	if hwid == "" {
		return
	}

	userUuid, err := dh.remnawaveClient.GetUserUuidByTelegramId(ctx, telegramId)
	if err != nil {
		slog.Error("Failed to get user UUID", "error", err)
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
			Text:            dh.translation.GetText(langCode, "device_delete_error"),
			ShowAlert:       true,
		})
		return
	}

	err = dh.remnawaveClient.DeleteUserDevice(ctx, userUuid, hwid)
	if err != nil {
		slog.Error("Failed to delete device", "error", err, "hwid", maskHwid(hwid))
		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
			Text:            dh.translation.GetText(langCode, "device_delete_error"),
			ShowAlert:       true,
		})
		return
	}

	slog.Info("Device deleted", "telegramId", utils.MaskHalfInt64(telegramId), "hwid", maskHwid(hwid))

	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callback.ID,
		Text:            dh.translation.GetText(langCode, "device_deleted"),
		ShowAlert:       false,
	})

	dh.DevicesCallbackHandler(ctx, b, update)
}
