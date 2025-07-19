package main

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/robfig/cron/v3"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"remnawave-tg-shop-bot/internal/cache"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/cryptopay"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/handler"
	"remnawave-tg-shop-bot/internal/notification"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/sync"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/internal/tribute"
	"remnawave-tg-shop-bot/internal/yookasa"
	"strconv"
	"strings"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config.InitConfig()

	tm := translation.GetInstance()
	err := tm.InitTranslations("./translations")
	if err != nil {
		panic(err)
	}

	pool, err := initDatabase(ctx, config.DadaBaseUrl())
	if err != nil {
		panic(err)
	}

	err = database.RunMigrations(ctx, &database.MigrationConfig{Direction: "up", MigrationsPath: "./db/migrations", Steps: 0}, pool)
	if err != nil {
		panic(err)
	}
	cache := cache.NewCache(30 * time.Minute)
	customerRepository := database.NewCustomerRepository(pool)
	purchaseRepository := database.NewPurchaseRepository(pool)
	referralRepository := database.NewReferralRepository(pool)
	promocodeRepository := database.NewPromocodeRepository(pool)

	cryptoPayClient := cryptopay.NewCryptoPayClient(config.CryptoPayUrl(), config.CryptoPayToken())
	remnawaveClient := remnawave.NewClient(config.RemnawaveUrl(), config.RemnawaveToken(), config.RemnawaveMode())
	yookasaClient := yookasa.NewClient(config.YookasaUrl(), config.YookasaShopId(), config.YookasaSecretKey())
	b, err := bot.New(config.TelegramToken(), bot.WithWorkers(3))
	if err != nil {
		panic(err)
	}

	paymentService := payment.NewPaymentService(tm, purchaseRepository, remnawaveClient, customerRepository, b, cryptoPayClient, yookasaClient, referralRepository, promocodeRepository, cache)

	cronScheduler := setupInvoiceChecker(purchaseRepository, cryptoPayClient, paymentService, yookasaClient)
	if cronScheduler != nil {
		cronScheduler.Start()
		defer cronScheduler.Stop()
	}

	subService := notification.NewSubscriptionService(customerRepository, b, tm)

	subscriptionNotificationCronScheduler := setupSubscriptionNotifier(subService)
	subscriptionNotificationCronScheduler.Start()
	defer subscriptionNotificationCronScheduler.Stop()

	syncService := sync.NewSyncService(remnawaveClient, customerRepository)

	h := handler.NewHandler(syncService, paymentService, tm, customerRepository, purchaseRepository, cryptoPayClient, yookasaClient, referralRepository, promocodeRepository, cache)

	me, err := b.GetMe(ctx)
	if err != nil {
		panic(err)
	}

	_, err = b.SetChatMenuButton(ctx, &bot.SetChatMenuButtonParams{
		MenuButton: &models.MenuButtonCommands{
			Type: models.MenuButtonTypeCommands,
		},
	})

	if err != nil {
		panic(err)
	}
	_, err = b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "start", Description: "Начать работу с ботом"},
		},
		LanguageCode: "ru",
	})

	_, err = b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "start", Description: "Start using the bot"},
		},
		LanguageCode: "en",
	})

	config.SetBotURL(fmt.Sprintf("https://t.me/%s", me.Username))

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, h.StartCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/connect", bot.MatchTypeExact, h.ConnectCommandHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/sync", bot.MatchTypeExact, h.SyncUsersCommandHandler, isAdminMiddleware)

	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackReferral, bot.MatchTypeExact, h.ReferralCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackPromoCreate, bot.MatchTypeExact, h.PromoCreateCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackBuy, bot.MatchTypeExact, h.BuyCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackTrial, bot.MatchTypeExact, h.TrialCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackActivateTrial, bot.MatchTypeExact, h.ActivateTrialCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackStart, bot.MatchTypeExact, h.StartCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackSell, bot.MatchTypePrefix, h.SellCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackConnect, bot.MatchTypeExact, h.ConnectCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackPayment, bot.MatchTypePrefix, h.PaymentCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackBalance, bot.MatchTypeExact, h.BalanceCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackTopup, bot.MatchTypeExact, h.TopupCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackTopupMethod, bot.MatchTypePrefix, h.TopupMethodCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackPayFromBal, bot.MatchTypePrefix, h.PayFromBalanceCallbackHandler, h.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.PreCheckoutQuery != nil
	}, h.PreCheckoutCallbackHandler, h.CreateCustomerIfNotExistMiddleware)

	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.SuccessfulPayment != nil
	}, h.SuccessPaymentHandler)

	mux := http.NewServeMux()
	mux.Handle("/healthcheck", fullHealthHandler(pool, remnawaveClient))
	if config.GetTributeWebHookUrl() != "" {
		tributeHandler := tribute.NewClient(paymentService, customerRepository)
		mux.Handle(config.GetTributeWebHookUrl(), tributeHandler.WebHookHandler())
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.GetHealthCheckPort()),
		Handler: mux,
	}
	go func() {
		log.Printf("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	slog.Info("Bot is starting...")
	b.Start(ctx)

	log.Println("Shutting down health server…")
	shutdownCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Health server shutdown error: %v", err)
	}
}

func fullHealthHandler(pool *pgxpool.Pool, rw *remnawave.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := map[string]string{
			"status": "ok",
			"db":     "ok",
			"rw":     "ok",
			"time":   time.Now().Format(time.RFC3339),
		}

		dbCtx, dbCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer dbCancel()
		if err := pool.Ping(dbCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			status["status"] = "fail"
			status["db"] = "error: " + err.Error()
		}

		rwCtx, rwCancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer rwCancel()
		if err := rw.Ping(rwCtx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			status["status"] = "fail"
			status["rw"] = "error: " + err.Error()
		}

		if status["status"] == "ok" {
			w.WriteHeader(http.StatusOK)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"%s","db":"%s","remnawave":"%s","time":"%s"}`,
			status["status"], status["db"], status["rw"], status["time"])
	})
}

func isAdminMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message != nil && config.IsAdmin(update.Message.From.ID) {
			next(ctx, b, update)
		} else {
			return
		}
	}
}

func setupSubscriptionNotifier(subService *notification.SubscriptionService) *cron.Cron {
	c := cron.New()

	_, err := c.AddFunc("0 16 * * *", func() {
		slog.Info("Running subscription notification check")

		err := subService.SendSubscriptionNotifications(context.Background())
		if err != nil {
			slog.Error("Error sending subscription notifications", "error", err)
		}
	})

	if err != nil {
		panic(err)
	}
	return c
}

func initDatabase(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 20
	config.MinConns = 5

	return pgxpool.ConnectConfig(ctx, config)
}

func setupInvoiceChecker(
	purchaseRepository *database.PurchaseRepository,
	cryptoPayClient *cryptopay.Client,
	paymentService *payment.PaymentService,
	yookasaClient *yookasa.Client) *cron.Cron {
	if !config.IsYookasaEnabled() && !config.IsCryptoPayEnabled() {
		return nil
	}
	c := cron.New(cron.WithSeconds())

	if config.IsCryptoPayEnabled() {
		_, err := c.AddFunc("*/5 * * * * *", func() {
			ctx := context.Background()
			checkCryptoPayInvoice(ctx, purchaseRepository, cryptoPayClient, paymentService)
		})

		if err != nil {
			panic(err)
		}
	}

	if config.IsYookasaEnabled() {
		_, err := c.AddFunc("*/5 * * * * *", func() {
			ctx := context.Background()
			checkYookasaInvoice(ctx, purchaseRepository, yookasaClient, paymentService)
		})

		if err != nil {
			panic(err)
		}
	}

	return c
}

func checkYookasaInvoice(
	ctx context.Context,
	purchaseRepository *database.PurchaseRepository,
	yookasaClient *yookasa.Client,
	paymentService *payment.PaymentService,
) {
	pendingPurchases, err := purchaseRepository.FindByInvoiceTypeAndStatus(
		ctx,
		database.InvoiceTypeYookasa,
		database.PurchaseStatusPending,
	)
	if err != nil {
		log.Printf("Error finding pending purchases: %v", err)
		return
	}
	if len(*pendingPurchases) == 0 {
		return
	}

	for _, purchase := range *pendingPurchases {

		invoice, err := yookasaClient.GetPayment(ctx, *purchase.YookasaID)

		if err != nil {
			slog.Error("Error getting invoice", "invoiceId", purchase.YookasaID, err)
			continue
		}

		if invoice.IsCancelled() {
			err := paymentService.CancelPayment(purchase.ID)
			if err != nil {
				slog.Error("Error canceling invoice", "invoiceId", invoice.ID, "purchaseId", purchase.ID, err)
			}
			continue
		}

		if !invoice.Paid {
			continue
		}

		purchaseId, err := strconv.Atoi(invoice.Metadata["purchaseId"])
		if err != nil {
			slog.Error("Error parsing purchaseId", "invoiceId", invoice.ID, err)
		}
		ctxWithValue := context.WithValue(ctx, "username", invoice.Metadata["username"])
		err = paymentService.ProcessPurchaseById(ctxWithValue, int64(purchaseId))
		if err != nil {
			slog.Error("Error processing invoice", "invoiceId", invoice.ID, "purchaseId", purchaseId, err)
		} else {
			slog.Info("Invoice processed", "invoiceId", invoice.ID, "purchaseId", purchaseId)
		}

	}
}

func checkCryptoPayInvoice(
	ctx context.Context,
	purchaseRepository *database.PurchaseRepository,
	cryptoPayClient *cryptopay.Client,
	paymentService *payment.PaymentService,
) {
	pendingPurchases, err := purchaseRepository.FindByInvoiceTypeAndStatus(
		ctx,
		database.InvoiceTypeCrypto,
		database.PurchaseStatusPending,
	)
	if err != nil {
		log.Printf("Error finding pending purchases: %v", err)
		return
	}
	if len(*pendingPurchases) == 0 {
		return
	}

	var invoiceIDs []string

	for _, purchase := range *pendingPurchases {
		if purchase.CryptoInvoiceID != nil {
			invoiceIDs = append(invoiceIDs, fmt.Sprintf("%d", *purchase.CryptoInvoiceID))
		}
	}

	if len(invoiceIDs) == 0 {
		return
	}

	stringInvoiceIDs := strings.Join(invoiceIDs, ",")
	invoices, err := cryptoPayClient.GetInvoices("", "", "", stringInvoiceIDs, 0, 0)
	if err != nil {
		log.Printf("Error getting invoices: %v", err)
		return
	}

	for _, invoice := range *invoices {
		if invoice.InvoiceID != nil && invoice.IsPaid() {
			payload := strings.Split(invoice.Payload, "&")
			purchaseID, err := strconv.Atoi(strings.Split(payload[0], "=")[1])
			username := strings.Split(payload[1], "=")[1]
			ctxWithUsername := context.WithValue(ctx, "username", username)
			err = paymentService.ProcessPurchaseById(ctxWithUsername, int64(purchaseID))
			if err != nil {
				slog.Error("Error processing invoice", "invoiceId", invoice.InvoiceID, err)
			} else {
				slog.Info("Invoice processed", "invoiceId", invoice.InvoiceID, "purchaseId", purchaseID)
			}

		}
	}
}
