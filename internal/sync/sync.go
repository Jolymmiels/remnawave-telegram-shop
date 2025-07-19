package sync

import (
	"context"
	"log/slog"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"
	"time"
)

type SyncService struct {
	client             *remnawave.Client
	customerRepository *database.CustomerRepository
}

func NewSyncService(client *remnawave.Client, customerRepository *database.CustomerRepository) *SyncService {
	return &SyncService{
		client: client, customerRepository: customerRepository,
	}
}

func (s SyncService) Sync() {
	slog.Info("Starting sync")
	ctx := context.Background()
	var telegramIDs []int64
	telegramIDsSet := make(map[int64]int64)
	var mappedUsers []database.Customer
	users, err := s.client.GetUsers(ctx)
	if err != nil {
		slog.Error("Error while getting users from remnawave")
		return
	}
	if users == nil || len(*users) == 0 {
		slog.Error("No users found in remnawave")
		return
	}

	for _, user := range *users {
		if user.TelegramId.Null {
			continue
		}
		if _, exists := telegramIDsSet[int64(user.TelegramId.Value)]; exists {
			continue
		}

		telegramIDsSet[int64(user.TelegramId.Value)] = int64(user.TelegramId.Value)

		telegramIDs = append(telegramIDs, int64(user.TelegramId.Value))

		mappedUsers = append(mappedUsers, database.Customer{
			TelegramID:       int64(user.TelegramId.Value),
			ExpireAt:         &user.ExpireAt,
			SubscriptionLink: &user.SubscriptionUrl,
			Balance:          0,
		})
	}

	existingCustomers, err := s.customerRepository.FindByTelegramIds(ctx, telegramIDs)
	if err != nil {
		slog.Error("Error while searching users by telegram ids")
		return
	}
	existingMap := make(map[int64]database.Customer)
	for _, cust := range existingCustomers {
		existingMap[cust.TelegramID] = cust
	}

	var toCreate []database.Customer
	var toUpdate []database.Customer

	for _, cust := range mappedUsers {
		if _, found := existingMap[cust.TelegramID]; found {
			cust.CreatedAt = time.Now()
			toUpdate = append(toUpdate, cust)
		} else {
			toCreate = append(toCreate, cust)
		}
	}

	err = s.customerRepository.DeleteByNotInTelegramIds(ctx, telegramIDs)
	if err != nil {
		slog.Error("Error while deleting users")
	}
	slog.Info("Deleted clients which not exist in panel")

	if len(toCreate) > 0 {
		if err := s.customerRepository.CreateBatch(ctx, toCreate); err != nil {
			slog.Error("Error while creating users")
		} else {
			slog.Info("Created clients", "count", len(toCreate))
		}
	}

	if len(toUpdate) > 0 {
		if err := s.customerRepository.UpdateBatch(ctx, toUpdate); err != nil {
			slog.Error("Error while updating users")
		} else {
			slog.Info("Updated clients", "count", len(toUpdate))
		}
	}
	slog.Info("Synchronization completed")
}
