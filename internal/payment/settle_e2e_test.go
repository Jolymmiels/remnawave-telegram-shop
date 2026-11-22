package payment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-telegram/bot"

	"remnawave-tg-shop-bot/internal/cache"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/database/dbtest"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/platega"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/translation"
)

const telegramID = 4242

type fakePanel struct {
	extensions atomic.Int32
}

func (p *fakePanel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := map[string]any{
		"id":              1,
		"username":        fmt.Sprintf("u_%d", telegramID),
		"subscriptionUrl": "https://sub.example/u",
		"expireAt":        time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		"telegramId":      telegramID,
		"status":          "ACTIVE",
	}
	var body any
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/users/stream"):
		body = map[string]any{"users": []any{user}, "hasMore": false}
	case r.Method == http.MethodGet && r.URL.Path == "/api/internal-squads":
		body = map[string]any{"internalSquads": []any{}}
	case r.Method == http.MethodPatch && r.URL.Path == "/api/users":
		time.Sleep(50 * time.Millisecond)
		p.extensions.Add(1)
		body = user
	default:
		http.NotFound(w, r)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"response": body})
}

type fakeTelegram struct {
	activations atomic.Int32
}

func (f *fakeTelegram) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/sendMessage") {
		f.activations.Add(1)
	}
	_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":4242,"type":"private"}}}`))
}

// Platega redelivers a callback it got no 200 for, and delivers duplicates
// concurrently. Each delivery of one payment must extend the subscription
// and announce it once in total.
func TestDuplicatePlategaCallbacksExtendSubscriptionOnce(t *testing.T) {
	pool := dbtest.Pool(t)
	ctx := context.Background()

	panel := &fakePanel{}
	panelServer := httptest.NewServer(panel)
	t.Cleanup(panelServer.Close)
	tg := &fakeTelegram{}
	tgServer := httptest.NewServer(tg)
	t.Cleanup(tgServer.Close)

	telegramBot, err := bot.New("1:test", bot.WithServerURL(tgServer.URL), bot.WithSkipGetMe())
	if err != nil {
		t.Fatal(err)
	}

	customers := database.NewCustomerRepository(pool)
	purchases := database.NewPurchaseRepository(pool)
	customer, err := customers.Create(ctx, &database.Customer{TelegramID: telegramID, Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	purchaseID, err := purchases.Create(ctx, &database.Purchase{
		InvoiceType: database.InvoiceTypePlatega,
		Status:      database.PurchaseStatusPending,
		Amount:      100,
		Currency:    "RUB",
		CustomerID:  customer.ID,
		Month:       1,
	})
	if err != nil {
		t.Fatal(err)
	}

	service := payment.NewPaymentService(
		translation.GetInstance(), purchases,
		remnawave.NewClient(panelServer.URL, "token", "remote"),
		customers, telegramBot, nil, nil, nil, nil,
		database.NewReferralRepository(pool), cache.NewCache(time.Minute), nil,
	)
	webhook := httptest.NewServer(platega.NewWebhookHandler(purchases, service, "merchant", "secret"))
	t.Cleanup(webhook.Close)

	deliver := func() int {
		body := fmt.Sprintf(`{"id":"tx-1","status":"CONFIRMED","payload":"purchaseId=%d"}`, purchaseID)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewBufferString(body))
		if err != nil {
			t.Error(err)
			return 0
		}
		req.Header.Set("X-MerchantId", "merchant")
		req.Header.Set("X-Secret", "secret")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return 0
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			if code := deliver(); code != http.StatusOK {
				t.Errorf("concurrent delivery got HTTP %d", code)
			}
		})
	}
	wg.Wait()
	if code := deliver(); code != http.StatusOK {
		t.Errorf("redelivery got HTTP %d", code)
	}

	if got := panel.extensions.Load(); got != 1 {
		t.Errorf("Remnawave got %d subscription extensions for one payment, want 1", got)
	}
	if got := tg.activations.Load(); got != 1 {
		t.Errorf("customer got %d activation messages for one payment, want 1", got)
	}
	settled, err := purchases.FindById(ctx, purchaseID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.Status != database.PurchaseStatusPaid || settled.PaidAt == nil {
		t.Errorf("purchase status %q paid_at %v, want paid with a time", settled.Status, settled.PaidAt)
	}
}
