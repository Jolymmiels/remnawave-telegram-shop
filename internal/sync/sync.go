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
	ctx := context.Background()
	pageSize := 100
	start := 0

	var allTelegramIDs []int64
	telegramIDsSet := make(map[int64]int64)

	for {
		usersResponse, err := s.client.GetUsers(ctx, pageSize, start)
		if err != nil {
			slog.Error("Error while getting users from remnawave")
			return
		}
		if usersResponse == nil || len(usersResponse.Users) == 0 {
			break
		}

		for _, user := range usersResponse.Users {
			if user.TelegramId == nil {
				continue
			}
			if _, exists := telegramIDsSet[*user.TelegramId]; exists {
				continue
			}

			telegramIDsSet[*user.TelegramId] = *user.TelegramId
			allTelegramIDs = append(allTelegramIDs, *user.TelegramId)
		}

		start += len(usersResponse.Users)
		if start >= usersResponse.Total {
			break
		}
	}

	start = 0
	for {
		usersResponse, err := s.client.GetUsers(ctx, pageSize, start)
		if err != nil {
			slog.Error("Error while getting users from remnawave")
			return
		}
		if usersResponse == nil || len(usersResponse.Users) == 0 {
			break
		}

		var mappedUsers []database.Customer
		for _, user := range usersResponse.Users {
			if user.TelegramId == nil {
				continue
			}

			customer, err := mapUserToCustomer(user)
			if err != nil {
				slog.Error("Error while mapping user from remnawave")
				continue
			}

			mappedUsers = append(mappedUsers, customer)
		}

		existingCustomers, err := s.customerRepository.FindByTelegramIds(ctx, allTelegramIDs)
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

		start += len(usersResponse.Users)
		if start >= usersResponse.Total {
			break
		}
	}

	err := s.customerRepository.DeleteByNotInTelegramIds(ctx, allTelegramIDs)
	if err != nil {
		slog.Error("Error while deleting users")
	} else {
		slog.Info("Deleted clients which not exist in panel")
	}

	slog.Info("Synchronization completed")
}

func mapUserToCustomer(user remnawave.User) (database.Customer, error) {
	return database.Customer{
		TelegramID:       *user.TelegramId,
		ExpireAt:         &user.ExpireAt,
		SubscriptionLink: &user.SubscriptionURL,
	}, nil
}
