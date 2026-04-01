package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"remnawave-tg-shop-bot/internal/cache"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/payment"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/telegramlink"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/internal/webauth"
	"remnawave-tg-shop-bot/internal/webserver"
	"remnawave-tg-shop-bot/internal/yookasa"
	"remnawave-tg-shop-bot/utils"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/robfig/cron/v3"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config.InitWebConfig()

	pool, err := initDatabase(ctx, config.DataBaseUrl())
	if err != nil {
		log.Fatal(err)
	}

	if err := database.RunMigrations(ctx, &database.MigrationConfig{
		Direction:      "up",
		MigrationsPath: "./db/migrations",
		Steps:          0,
	}, pool); err != nil {
		log.Fatal(err)
	}

	remnawaveClient := remnawave.NewClient(config.RemnawaveUrl(), config.RemnawaveToken(), config.RemnawaveMode())
	customerRepository := database.NewCustomerRepository(pool)
	purchaseRepository := database.NewPurchaseRepository(pool)
	telegramLinkRepository := database.NewTelegramLinkRepository(pool)
	authService := webauth.NewService(customerRepository, config.WebSessionSecret())
	tm := translation.GetInstance()
	if err := tm.InitTranslations("./translations", config.DefaultLanguage()); err != nil {
		log.Fatal(err)
	}
	yookasaClient := yookasa.NewClient(config.YookasaUrl(), config.YookasaShopId(), config.YookasaSecretKey())
	paymentService := payment.NewPaymentService(
		tm,
		purchaseRepository,
		remnawaveClient,
		customerRepository,
		nil,
		nil,
		yookasaClient,
		nil,
		cache.NewCache(30*time.Minute),
		nil,
	)
	telegramLinkService := telegramlink.NewService(customerRepository, telegramLinkRepository)

	invoiceChecker := setupYookassaInvoiceChecker(purchaseRepository, yookasaClient, paymentService)
	invoiceChecker.Start()
	defer invoiceChecker.Stop()

	server := webserver.New(pool, remnawaveClient, customerRepository, purchaseRepository, authService, paymentService, yookasaClient, telegramLinkService)
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", config.WebPort()),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("Web API listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("web server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("web shutdown error", "error", err)
	}
}

func initDatabase(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	poolConfig.MaxConns = 20
	poolConfig.MinConns = 5

	return pgxpool.ConnectConfig(ctx, poolConfig)
}

func setupYookassaInvoiceChecker(
	purchaseRepository *database.PurchaseRepository,
	yookasaClient *yookasa.Client,
	paymentService *payment.PaymentService,
) *cron.Cron {
	c := cron.New(cron.WithSeconds())

	_, err := c.AddFunc("*/5 * * * * *", func() {
		checkYookasaInvoice(context.Background(), purchaseRepository, yookasaClient, paymentService)
	})
	if err != nil {
		panic(err)
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
	if err != nil || len(*pendingPurchases) == 0 {
		return
	}

	for _, purchase := range *pendingPurchases {
		invoice, err := yookasaClient.GetPayment(ctx, *purchase.YookasaID)
		if err != nil {
			slog.Error("web yookassa: get payment", "error", err, "purchase_id", purchase.ID)
			continue
		}

		if invoice.IsCancelled() {
			if err := paymentService.CancelYookassaPayment(purchase.ID); err != nil {
				slog.Error("web yookassa: cancel payment", "error", err, "purchase_id", purchase.ID)
			}
			continue
		}

		if !invoice.Paid {
			continue
		}

		purchaseID, err := strconv.Atoi(invoice.Metadata["purchaseId"])
		if err != nil {
			slog.Error("web yookassa: parse purchaseId", "error", err, "invoice_id", invoice.ID)
			continue
		}

		ctxWithUsername := context.WithValue(ctx, utils.ContextKeyUsername, invoice.Metadata["username"])
		if err := paymentService.ProcessPurchaseById(ctxWithUsername, int64(purchaseID)); err != nil {
			slog.Error("web yookassa: process purchase", "error", err, "purchase_id", purchaseID)
		}
	}
}
