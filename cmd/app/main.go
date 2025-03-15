package main

import (
	"context"
	"fmt"
	"github.com/biter777/countries"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/robfig/cron/v3"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/cryptopay"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/handler"
	"remnawave-tg-shop-bot/internal/notification"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/internal/yookasa"
	"strconv"
	"strings"
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

	customerRepository := database.NewCustomerRepository(pool)
	purchaseRepository := database.NewPurchaseRepository(pool)

	cryptoPayClient := cryptopay.NewCryptoPayClient(config.CryptoPayUrl(), config.CryptoPayToken())
	remnawaveClient := remnawave.NewClient(config.RemnawaveUrl(), config.RemnawaveToken())
	initCountries(ctx, remnawaveClient)
	yookasaClient := yookasa.NewClient(config.YookasaUrl(), config.YookasaShopId(), config.YookasaSecretKey())
	b, err := bot.New(config.TelegramToken(), bot.WithWorkers(3))
	if err != nil {
		panic(err)
	}

	paymentService := payment.NewPaymentService(tm, purchaseRepository, remnawaveClient, customerRepository, b)

	cronScheduler := setupInvoiceChecker(purchaseRepository, cryptoPayClient, paymentService, yookasaClient)
	if cronScheduler != nil {
		cronScheduler.Start()
		defer cronScheduler.Stop()
	}

	subService := notification.NewSubscriptionService(customerRepository, b, tm)

	subscriptionNotificationCronScheduler := setupSubscriptionNotifier(subService)
	subscriptionNotificationCronScheduler.Start()
	defer subscriptionNotificationCronScheduler.Stop()

	h := handler.NewHandler(paymentService, tm, customerRepository, purchaseRepository, cryptoPayClient, yookasaClient)

	me, err := b.GetMe(ctx)
	if err != nil {
		panic(err)
	}

	_, err = b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "start", Description: "Начать работу с ботом"},
			{Command: "connect", Description: "Подключиться к VPN"},
		},
		LanguageCode: "ru",
	})

	if err != nil {
		panic(err)
	}

	_, err = b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "start", Description: "Start using the bot"},
			{Command: "connect", Description: "Connect to VPN"},
		},
		LanguageCode: "en",
	})

	if err != nil {
		panic(err)
	}

	config.SetBotURL(fmt.Sprintf("https://t.me/%s", me.Username))

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, h.StartCommandHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/connect", bot.MatchTypeExact, h.ConnectCommandHandler)

	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackBuy, bot.MatchTypeExact, h.BuyCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackStart, bot.MatchTypeExact, h.StartCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackSell, bot.MatchTypePrefix, h.SellCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackCrypto, bot.MatchTypePrefix, h.CryptoCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackConnect, bot.MatchTypeExact, h.ConnectCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackCard, bot.MatchTypePrefix, h.YookasaCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackTelegramStars, bot.MatchTypePrefix, h.TelegramStarsCallbackHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, handler.CallbackTrial, bot.MatchTypeExact, h.TrialCallbackHandler)

	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.PreCheckoutQuery != nil
	}, h.PreCheckoutCallbackHandler)

	b.RegisterHandlerMatchFunc(func(update *models.Update) bool {
		return update.Message != nil && update.Message.SuccessfulPayment != nil
	}, h.SuccessPaymentHandler)

	slog.Info("Bot is starting...")
	b.Start(ctx)
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

func initCountries(ctx context.Context, remnawaveClient *remnawave.Client) {
	nodes, err := remnawaveClient.GetNodes(ctx)
	if err != nil {
		panic("error getting nodes")
	}

	uniqueCountries := make(map[string]string)

	for _, node := range *nodes {
		if !node.IsDisabled && node.IsNodeOnline {
			country := countries.ByName(node.CountryCode)
			countryText := fmt.Sprintf("%s %s", country.Emoji(), node.CountryCode)
			uniqueCountries[node.CountryCode] = countryText
		}
	}

	config.SetCountries(uniqueCountries)

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
		}
		if !invoice.Paid {
			continue
		}
		purchaseId, err := strconv.Atoi(invoice.Metadata["purchaseId"])
		if err != nil {
			slog.Error("Error parsing purchaseId", "invoiceId", invoice.ID, err)
		}

		err = paymentService.ProcessPurchaseById(int64(purchaseId))
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
			purchaseParam := strings.Split(invoice.Payload, "&")[1]
			purchaseID, err := strconv.Atoi(strings.Split(purchaseParam, "=")[1])
			err = paymentService.ProcessPurchaseById(int64(purchaseID))
			if err != nil {
				slog.Error("Error processing invoice", "invoiceId", invoice.InvoiceID, err)
			} else {
				slog.Info("Invoice processed", "invoiceId", invoice.InvoiceID, "purchaseId", purchaseID)
			}

		}
	}
}
