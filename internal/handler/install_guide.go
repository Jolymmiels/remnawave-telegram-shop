package handler

import (
	"context"

	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (h Handler) InstallGuideCallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	langCode := update.CallbackQuery.From.LanguageCode
	callbackMessage := update.CallbackQuery.Message.Message

	callbackQuery := parseCallbackData(update.CallbackQuery.Data)
	backCallback := callbackQuery["from"]

	if backCallback == "" {
		backCallback = CallbackStart
	}

	keyboard := [][]models.InlineKeyboardButton{
		{
			{Text: h.translation.GetText(langCode, "ios_button"), URL: "https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973"},
		},
		{
			{Text: h.translation.GetText(langCode, "android_button"), URL: "https://play.google.com/store/apps/details?id=com.happproxy"},
		},
		{
			{Text: h.translation.GetText(langCode, "windows_button"), URL: "https://github.com/Happ-proxy/happ-desktop/releases/latest/download/setup-Happ.x86.exe"},
		},
		{
			{Text: h.translation.GetText(langCode, "macos_button"), URL: "https://apps.apple.com/ru/app/happ-proxy-utility-plus/id6746188973"},
		},
		{
			{Text: h.translation.GetText(langCode, "linux_button"), URL: "https://github.com/Happ-proxy/happ-desktop/releases/latest/download/Happ.linux.x86.AppImage.zip"},
		},
		{
			{Text: h.translation.GetText(langCode, "back_button"), CallbackData: backCallback},
		},
	}

	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      callbackMessage.Chat.ID,
		MessageID:   callbackMessage.ID,
		Text:        h.translation.GetText(langCode, "install_guide_title") + "\n\n" + h.translation.GetText(langCode, "install_guide_text"),
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: keyboard},
	})
	if err != nil {
		slog.Error("Error sending referral message", err)
	}
}
