package handler

import (
	"encoding/json"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"strings"
)

type BotConfigResponse struct {
	BotUsername string `json:"bot_username"`
}

func GetBotConfig(w http.ResponseWriter, r *http.Request) {
	botURL := config.BotURL()
	botUsername := ""
	
	// Extract username from URL like "https://t.me/botname"
	if strings.Contains(botURL, "t.me/") {
		parts := strings.Split(botURL, "t.me/")
		if len(parts) > 1 {
			botUsername = parts[1]
		}
	}

	response := BotConfigResponse{
		BotUsername: botUsername,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
