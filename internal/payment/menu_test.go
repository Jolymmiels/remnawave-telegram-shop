package payment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"remnawave-tg-shop-bot/internal/cache"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/translation"
)

func TestPaymentKeepsMenuMessageAndPhoto(t *testing.T) {
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			return
		}
		defer r.MultipartForm.RemoveAll()
		requests <- r.URL.Path
		if r.FormValue("message_id") != "99" || r.FormValue("chat_id") != "42" {
			t.Error("wrong menu edited")
		}
		var markup models.InlineKeyboardMarkup
		if err := json.Unmarshal([]byte(r.FormValue("reply_markup")), &markup); err != nil {
			t.Error(err)
		}
		if len(markup.InlineKeyboard) != 2 || markup.InlineKeyboard[1][0].CallbackData != "start" {
			t.Error("lost menu navigation")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": models.Message{ID: 99}})
	}))
	defer server.Close()
	b, err := bot.New("123:test", bot.WithSkipGetMe(), bot.WithServerURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	c := cache.NewCache(time.Minute)
	c.Set(1, 99)
	s := PaymentService{telegramBot: b, cache: c, translation: translation.GetInstance()}
	s.refreshPurchaseMenu(context.Background(), 1, &database.Customer{TelegramID: 42, Language: "en"})
	if path := <-requests; !strings.HasSuffix(path, "/editMessageReplyMarkup") {
		t.Fatalf("menu was not preserved: %s", path)
	}
	s.refreshPurchaseMenu(context.Background(), 2, &database.Customer{TelegramID: 42})
	select {
	case path := <-requests:
		t.Fatalf("uncached purchase touched menu: %s", path)
	default:
	}
}
