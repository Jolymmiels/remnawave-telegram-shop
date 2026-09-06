package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"remnawave-tg-shop-bot/internal/translation"
)

func TestBuyCallbackEditsCaptionForPhoto(t *testing.T) {
	for _, photo := range []bool{false, true} {
		method := "editMessageText"
		if photo {
			method = "editMessageCaption"
		}
		t.Run(method, func(t *testing.T) {
			requests := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				requests <- r.URL.Path
				if r.FormValue("message_id") != "99" || !strings.Contains(r.FormValue("reply_markup"), "start") {
					t.Error("lost menu navigation")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{\"ok\":true,\"result\":{\"message_id\":99}}"))
			}))
			defer server.Close()
			b, err := bot.New("123:test", bot.WithSkipGetMe(), bot.WithServerURL(server.URL))
			if err != nil {
				t.Fatal(err)
			}
			msg := &models.Message{ID: 99, Chat: models.Chat{ID: 42}}
			if photo {
				msg.Photo = []models.PhotoSize{{FileID: "keep-photo"}}
			}
			h := Handler{translation: translation.GetInstance()}
			h.BuyCallbackHandler(context.Background(), b, &models.Update{CallbackQuery: &models.CallbackQuery{
				From: models.User{ID: 42, LanguageCode: "en"}, Message: models.MaybeInaccessibleMessage{Message: msg},
			}})
			if path := <-requests; !strings.HasSuffix(path, "/"+method) {
				t.Fatalf("wrong API method: %s", path)
			}
		})
	}
}
