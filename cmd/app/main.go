package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"remnawave-tg-shop-bot/internal/broadcast"
	"remnawave-tg-shop-bot/internal/cache"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/cryptopay"
	"remnawave-tg-shop-bot/internal/database"
	tghandler "remnawave-tg-shop-bot/internal/handler"
	httpserver "remnawave-tg-shop-bot/internal/http"
	httphandler "remnawave-tg-shop-bot/internal/http/handler"
	"remnawave-tg-shop-bot/internal/notification"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/promo"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/sync"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/internal/yookasa"
	"remnawave-tg-shop-bot/utils"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/robfig/cron/v3"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := config.InitConfig(); err != nil {
		log.Fatal("Failed to initialize config: ", err)
	}
	slog.Info("Application starting", "version", Version, "commit", Commit, "buildDate", BuildDate)

	tm := translation.GetInstance()
	err := tm.InitTranslations("./translations", config.DefaultLanguage())
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
	settingsRepository := database.NewSettingsRepository(pool)
	if err := settingsRepository.LoadAll(ctx); err != nil {
		slog.Warn("Failed to load settings from database, using env defaults", "error", err)
	}
	// Set settings provider for config to use DB settings with env fallback
	config.SetSettingsProvider(settingsRepository)
	planRepository := database.NewPlanRepository(pool)

	cryptoPayClient := cryptopay.NewCryptoPayClient()
	remnawaveClient, err := remnawave.NewClient(config.RemnawaveUrl(), config.RemnawaveToken(), config.RemnawaveMode())
	if err != nil {
		log.Fatal("Failed to create remnawave client: ", err)
	}
	yookasaClient := yookasa.NewClient()
	b, err := bot.New(config.TelegramToken(), bot.WithWorkers(3))
	if err != nil {
		log.Fatal("Failed to create telegram bot: ", err)
	}

	paymentService := payment.NewPaymentService(tm, purchaseRepository, remnawaveClient, customerRepository, b, cryptoPayClient, yookasaClient, referralRepository, cache, planRepository, settingsRepository)

	cronScheduler := setupInvoiceChecker(purchaseRepository, cryptoPayClient, paymentService, yookasaClient)
	if cronScheduler != nil {
		cronScheduler.Start()
		defer cronScheduler.Stop()
	}

	subService := notification.NewSubscriptionService(customerRepository, purchaseRepository, paymentService, b, tm)

	subscriptionNotificationCronScheduler := subscriptionChecker(subService)
	subscriptionNotificationCronScheduler.Start()
	defer subscriptionNotificationCronScheduler.Stop()

	// Autopayment cron scheduler
	autopaymentCronScheduler := setupAutopaymentChecker(paymentService)
	autopaymentCronScheduler.Start()
	defer autopaymentCronScheduler.Stop()

	syncService := sync.NewSyncService(remnawaveClient, customerRepository)

	promoService := promo.NewService(database.NewPromoRepository(pool))
	broadcastRepository := database.NewBroadcastRepository(pool)
	broadcastService := broadcast.NewService(broadcastRepository, b, customerRepository)
	broadcastService.SetAppContext(ctx)
	statsQueries := database.NewStatsQueries(pool)
	statsHandler := httphandler.NewStatsHandler(purchaseRepository, customerRepository, statsQueries)

	// Create domain-specific handlers
	middleware := tghandler.NewMiddlewareHandler(customerRepository, tm)
	startHandler := tghandler.NewStartHandler(customerRepository, referralRepository, promoService, tm)
	paymentHandler := tghandler.NewPaymentHandler(customerRepository, purchaseRepository, planRepository, settingsRepository, paymentService, tm, cache)
	connectHandler := tghandler.NewConnectHandler(customerRepository, tm, remnawaveClient)
	trialHandler := tghandler.NewTrialHandler(customerRepository, paymentService, tm)
	referralHandler := tghandler.NewReferralHandler(customerRepository, referralRepository, tm)
	syncHandler := tghandler.NewSyncHandler(syncService)
	promoHandler := tghandler.NewPromoHandler(customerRepository, promoService, paymentService, tm)
	adminHandler := tghandler.NewAdminHandler()
	devicesHandler := tghandler.NewDevicesHandler(remnawaveClient, customerRepository, purchaseRepository, planRepository, tm)
	autopayHandler := tghandler.NewAutopayHandler(customerRepository, paymentService, tm)

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

	// Register command handlers
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypePrefix, startHandler.StartCommandHandler, middleware.SuspiciousUserFilterMiddleware)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/connect", bot.MatchTypeExact, connectHandler.ConnectCommandHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/sync", bot.MatchTypeExact, syncHandler.SyncUsersCommandHandler, isAdminMiddleware)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/admin", bot.MatchTypeExact, adminHandler.AdminCommandHandler, isAdminMiddleware)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/promo", bot.MatchTypePrefix, promoHandler.PromoCommandHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)

	// Register callback handlers
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackReferral, bot.MatchTypeExact, referralHandler.ReferralCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackBuy, bot.MatchTypeExact, paymentHandler.BuyCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackPlan, bot.MatchTypePrefix, paymentHandler.PlanCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackTrial, bot.MatchTypeExact, trialHandler.TrialCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackActivateTrial, bot.MatchTypeExact, trialHandler.ActivateTrialCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackStart, bot.MatchTypeExact, startHandler.StartCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackSell, bot.MatchTypePrefix, paymentHandler.SellCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackConnect, bot.MatchTypeExact, connectHandler.ConnectCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackPaymentMethods, bot.MatchTypeExact, autopayHandler.PaymentMethodsCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackPayment, bot.MatchTypePrefix, paymentHandler.PaymentCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackPromo, bot.MatchTypePrefix, promoHandler.PromoCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)

	// Devices handlers
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackDevices, bot.MatchTypeExact, devicesHandler.DevicesCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackDeviceDelete, bot.MatchTypePrefix, devicesHandler.DeviceDeleteCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackDeviceConfirm, bot.MatchTypePrefix, devicesHandler.DeviceConfirmDeleteHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)

	// Autopay handler
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackAutopayDisable, bot.MatchTypeExact, autopayHandler.AutopayDisableCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackAutopayEnable, bot.MatchTypeExact, autopayHandler.AutopayEnableCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, tghandler.CallbackDeletePayment, bot.MatchTypeExact, autopayHandler.DeletePaymentMethodCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)

	// Payment flow handlers
	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.PreCheckoutQuery != nil
	}, paymentHandler.PreCheckoutCallbackHandler, middleware.SuspiciousUserFilterMiddleware, middleware.CreateCustomerIfNotExistMiddleware)

	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.SuccessfulPayment != nil
	}, paymentHandler.SuccessPaymentHandler, middleware.SuspiciousUserFilterMiddleware)

	srv := httpserver.NewServer(statsHandler, pool, remnawaveClient, paymentService, broadcastService, promoService, customerRepository, purchaseRepository, referralRepository, syncService, settingsRepository, planRepository)
	go func() {
		log.Printf("HTTP server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	slog.Info("Bot is starting...")
	b.Start(ctx)

	log.Println("Shutting down HTTP server…")
	shutdownCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
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

func subscriptionChecker(subService *notification.SubscriptionService) *cron.Cron {
	c := cron.New()

	_, err := c.AddFunc("0 16 * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		err := subService.ProcessSubscriptionExpiration(ctx)
		if err != nil {
			slog.Error("Error sending subscription notifications", "error", err)
		}
	})

	if err != nil {
		log.Fatal("Failed to add subscription checker cron job: ", err)
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
	c := cron.New(cron.WithSeconds())

	// CryptoPay checker - checks setting on each tick
	_, err := c.AddFunc("*/5 * * * * *", func() {
		if !config.IsCryptoPayEnabled() {
			return
		}
		ctx := context.Background()
		checkCryptoPayInvoice(ctx, purchaseRepository, cryptoPayClient, paymentService)
	})
	if err != nil {
		log.Fatal("Failed to add CryptoPay invoice checker: ", err)
	}

	// YooKassa checker - checks setting on each tick
	_, err = c.AddFunc("*/5 * * * * *", func() {
		if !config.IsYookasaEnabled() {
			return
		}
		ctx := context.Background()
		checkYookasaInvoice(ctx, purchaseRepository, yookasaClient, paymentService)
	})
	if err != nil {
		log.Fatal("Failed to add YooKassa invoice checker: ", err)
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
		database.InvoiceTypeYookassa,
		database.PurchaseStatusPending,
	)
	if err != nil {
		log.Printf("Error finding pending purchases: %v", err)
		return
	}
	if len(pendingPurchases) == 0 {
		return
	}

	for _, purchase := range pendingPurchases {

		invoice, err := yookasaClient.GetPayment(ctx, *purchase.YookasaID)

		if err != nil {
			slog.Error("Error getting invoice", "invoiceId", purchase.YookasaID, "error", err)
			continue
		}

		if invoice.IsCancelled() {
			err := paymentService.CancelYookassaPayment(purchase.ID)
			if err != nil {
				slog.Error("Error canceling invoice", "invoiceId", invoice.ID, "purchaseId", purchase.ID, "error", err)
			}
			continue
		}

		if !invoice.Paid {
			continue
		}

		purchaseId, err := strconv.Atoi(invoice.Metadata["purchaseId"])
		if err != nil {
			slog.Error("Error parsing purchaseId", "invoiceId", invoice.ID, "error", err)
		}
		ctxWithValue := context.WithValue(ctx, "username", invoice.Metadata["username"])
		err = paymentService.ProcessPurchaseById(ctxWithValue, int64(purchaseId))
		if err != nil {
			slog.Error("Error processing invoice", "invoiceId", invoice.ID, "purchaseId", purchaseId, "error", err)
		} else {
			slog.Info("Invoice processed", "invoiceId", utils.MaskHalf(invoice.ID.String()), "purchaseId", purchaseId)

			// Save payment method if it was saved for autopay
			if savedPaymentMethodID := invoice.GetSavedPaymentMethodID(); savedPaymentMethodID != nil {
				customerIdStr := invoice.Metadata["customerId"]
				customerId, parseErr := strconv.ParseInt(customerIdStr, 10, 64)
				if parseErr == nil && purchase.PlanID != nil {
					saveErr := paymentService.SavePaymentMethod(ctx, customerId, savedPaymentMethodID.String(), *purchase.PlanID, purchase.Month)
					if saveErr != nil {
						slog.Error("Error saving payment method", "customerId", customerId, "error", saveErr)
					} else {
						slog.Info("Payment method saved for autopay", "customerId", customerId, "paymentMethodId", utils.MaskHalf(savedPaymentMethodID.String()))
					}
				}
			}
		}

	}
}

func setupAutopaymentChecker(paymentService *payment.PaymentService) *cron.Cron {
	c := cron.New(cron.WithSeconds())

	_, err := c.AddFunc("30 * * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := paymentService.ProcessAutopayments(ctx); err != nil {
			slog.Error("Error processing autopayments", "error", err)
		}
	})
	if err != nil {
		log.Fatal("Failed to add autopayment checker cron job: ", err)
	}

	// Run autopayment notification every day at 10:00
	_, err = c.AddFunc("0 10 * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := paymentService.NotifyUpcomingAutopayments(ctx); err != nil {
			slog.Error("Error sending autopayment notifications", "error", err)
		}
	})
	if err != nil {
		log.Fatal("Failed to add autopayment notification cron job: ", err)
	}

	return c
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
	if len(pendingPurchases) == 0 {
		return
	}

	var invoiceIDs []string

	for _, purchase := range pendingPurchases {
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
				slog.Error("Error processing invoice", "invoiceId", invoice.InvoiceID, "error", err)
			} else {
				slog.Info("Invoice processed", "invoiceId", invoice.InvoiceID, "purchaseId", purchaseID)
			}

		}
	}
}
