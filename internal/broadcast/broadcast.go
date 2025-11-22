package broadcast

import (
	"context"
	"github.com/go-telegram/bot"
	"log/slog"
	"private-remnawave-telegram-shop-bot/internal/database"
)

type Service struct {
	repo      *database.BroadcastRepository
	customers *database.CustomerRepository
	tgBotApi  *bot.Bot
}

func NewService(repo *database.BroadcastRepository, tgBotApi *bot.Bot, customers *database.CustomerRepository) *Service {
	return &Service{repo: repo, tgBotApi: tgBotApi, customers: customers}
}

func (s *Service) CreateBroadcast(ctx context.Context, content, broadcastType, language string) (*database.Broadcast, error) {

	br := &database.Broadcast{
		Content:  content,
		Type:     broadcastType,
		Language: language,
	}

	created, err := s.repo.CreateBroadcast(ctx, br)
	if err != nil {
		slog.Error("failed to create broadcast", "error", err)
		return nil, err
	}

	var customers *[]database.Customer
	switch broadcastType {
	case database.BroadcastAll:
		customers, err = s.customers.FindAll(ctx)
	case database.BroadcastActive:
		customers, err = s.customers.FindNonExpired(ctx)
	case database.BroadcastInactive:
		customers, err = s.customers.FindExpired(ctx)
	}
	if err != nil {
		slog.Error("failed to find customers", "error", err)
		return nil, err
	}

	if customers == nil {
		return created, nil
	}

	if len(*customers) == 0 {
		return created, nil
	}

	for _, customer := range *customers {
		_, err = s.tgBotApi.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: customer.TelegramID,
			Text:   content,
		})
		if err != nil {
			slog.Error("failed to send message", "error", err)
		}
	}

	return created, err

}

func (s *Service) List(ctx context.Context, params database.BroadcastListParams) (*[]database.Broadcast, error) {
	return s.repo.List(ctx, params)
}
