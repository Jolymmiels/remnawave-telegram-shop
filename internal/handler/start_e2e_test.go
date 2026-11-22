package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"remnawave-tg-shop-bot/internal/cache"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/database/dbtest"
	"remnawave-tg-shop-bot/internal/handler"
	"remnawave-tg-shop-bot/internal/menu"
	"remnawave-tg-shop-bot/internal/translation"
)

type fakeTelegram struct {
	mu    sync.Mutex
	texts []string
}

func (f *fakeTelegram) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	if method == "sendMessage" {
		_ = r.ParseMultipartForm(1 << 20)
		f.mu.Lock()
		f.texts = append(f.texts, r.FormValue("text"))
		f.mu.Unlock()
	}
	result := `{"message_id":7,"date":0,"chat":{"id":4242,"type":"private"}}`
	if method != "sendMessage" {
		result = "true"
	}
	_, _ = w.Write([]byte(`{"ok":true,"result":` + result + `}`))
}

func (f *fakeTelegram) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.texts...)
}

// A newcomer whose referral link names someone who never used the bot must
// still get the start menu; the referral is just not recorded.
func TestStartWithUnknownReferrerShowsMenu(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()

	tm := translation.GetInstance()
	if err := tm.InitTranslations("../../translations", "en"); err != nil {
		t.Fatal(err)
	}
	tg := &fakeTelegram{}
	server := httptest.NewServer(tg)
	t.Cleanup(server.Close)
	b, err := bot.New("1:test", bot.WithServerURL(server.URL), bot.WithSkipGetMe())
	if err != nil {
		t.Fatal(err)
	}
	menuService, err := menu.New(ctx, database.NewMenuRepository(pool), 1, "")
	if err != nil {
		t.Fatal(err)
	}
	customers := database.NewCustomerRepository(pool)
	referrals := database.NewReferralRepository(pool)
	h := handler.NewHandler(nil, nil, tm, customers, database.NewPurchaseRepository(pool), nil, nil, referrals, cache.NewCache(time.Minute), menuService)

	h.StartCommandHandler(ctx, b, &models.Update{Message: &models.Message{
		Text: "/start ref_999",
		Chat: models.Chat{ID: 4242},
		From: &models.User{ID: 4242, LanguageCode: "en"},
	}})

	greeting := tm.GetText("en", "greeting")
	texts := tg.sent()
	if len(texts) == 0 || texts[len(texts)-1] != greeting {
		got, _ := json.Marshal(texts)
		t.Fatalf("newcomer did not get the start menu; sent messages %s", got)
	}
	if _, err := referrals.FindByReferee(ctx, 4242); err == nil {
		t.Error("recorded a referral from a referrer who is not a customer")
	}
}
