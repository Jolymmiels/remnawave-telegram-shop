package payment

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/internal/translation"
	"remnawave-tg-shop-bot/internal/config"
	"time"
)

type PaymentService struct {
	purchaseRepository *database.PurchaseRepository
	remnawaveClient    *remnawave.Client
	customerRepository *database.CustomerRepository
	telegramBot        *bot.Bot
	translation        *translation.Manager
}

func NewPaymentService(
	translation *translation.Manager,
	purchaseRepository *database.PurchaseRepository,
	remnawaveClient *remnawave.Client,
	customerRepository *database.CustomerRepository,
	telegramBot *bot.Bot) *PaymentService {
	return &PaymentService{
		purchaseRepository: purchaseRepository,
		remnawaveClient:    remnawaveClient,
		customerRepository: customerRepository,
		telegramBot:        telegramBot,
		translation:        translation,
	}
}

func (s PaymentService) ProcessPurchaseById(purchaseId int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()
	purchase, err := s.purchaseRepository.FindById(ctx, purchaseId)
	if err != nil {
		return err
	}
	if purchase == nil {
		return fmt.Errorf("purchase with crypto invoice id %d not found", purchaseId)
	}

	customer, err := s.customerRepository.FindById(ctx, purchase.CustomerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return fmt.Errorf("customer %s not found", purchase.CustomerID)
	}

	username := fmt.Sprintf("%d_%d", customer.ID, customer.TelegramID)

	user, err := s.remnawaveClient.CreateOrUpdateUser(ctx, username, purchase.Month, config.TrafficLimit())
	if err != nil {
		return err
	}

	err = s.purchaseRepository.MarkAsPaid(ctx, purchase.ID)
	if err != nil {
		return err
	}

	customerFilesToUpdate := map[string]interface{}{
		"subscription_link": user.SubscriptionURL,
		"expire_at":         user.ExpireAt,
	}

	err = s.customerRepository.UpdateFields(ctx, customer.ID, customerFilesToUpdate)
	if err != nil {
		return err
	}

	_, err = s.telegramBot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: customer.TelegramID,
		Text:   s.translation.GetText(customer.Language, "subscription_activated"),
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: s.translation.GetText(customer.Language, "connect_button"), URL: user.SubscriptionURL},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}


func (s PaymentService) ActivateTrialSubscription(customerID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
	defer cancel()

	customer, err := s.customerRepository.FindByTelegramId(ctx, customerID)
	if err != nil {
		return err
	}
	if customer == nil {
		return fmt.Errorf("customer with id %d not found", customerID)
	}

	if customer.SubscriptionLink != nil && *customer.SubscriptionLink != ""{
		return fmt.Errorf("пробная подписка уже активирована или исчерпана")
	}

	username := fmt.Sprintf("%d_%d", customer.ID, customer.TelegramID)
	month  := 1
	user, err := s.remnawaveClient.CreateOrUpdateUser(ctx, username, month, config.TrialTrafficLimitGB())
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"subscription_link":  user.SubscriptionURL,
		"expire_at":          user.ExpireAt,
	}

	err = s.customerRepository.UpdateFields(ctx, customer.ID, updates)
	if err != nil {
		return err
	}


	return nil
}
