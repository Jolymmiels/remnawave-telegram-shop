package sync

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/internal/remnawave"
	"remnawave-tg-shop-bot/utils"
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
	mappedUserIndexes := make(map[int64]int)
	usersForSync := make(map[int64]remnawave.User)
	var mappedUsers []database.Customer
	users, err := s.client.GetUsers(ctx)
	if err != nil {
		slog.Error("Error while getting users from remnawave", "error", err)
		return
	}
	if len(users) == 0 {
		slog.Error("No users found in remnawave")
		return
	}

	for _, user := range users {
		if user.TelegramID == nil {
			continue
		}
		tid := *user.TelegramID
		if index, exists := mappedUserIndexes[tid]; exists {
			current := usersForSync[tid]
			preferred := preferManagedUser(current, user, tid)
			usersForSync[tid] = preferred
			mappedUsers[index].ExpireAt = &preferred.ExpireAt
			mappedUsers[index].SubscriptionLink = &preferred.SubscriptionUrl
			slog.Warn("Multiple panel users have the same Telegram ID; selected one subscription", "telegramId", utils.MaskHalfInt64(tid), "selectedUserId", preferred.ID)
			continue
		}

		mappedUserIndexes[tid] = len(mappedUsers)
		usersForSync[tid] = user
		telegramIDs = append(telegramIDs, tid)

		mappedUsers = append(mappedUsers, database.Customer{
			TelegramID:       tid,
			ExpireAt:         &user.ExpireAt,
			SubscriptionLink: &user.SubscriptionUrl,
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
		if existing, found := existingMap[cust.TelegramID]; found {
			cust.ID = existing.ID
			cust.CreatedAt = existing.CreatedAt
			cust.Language = existing.Language
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

func preferManagedUser(current, candidate remnawave.User, telegramID int64) remnawave.User {
	suffix := "_" + strconv.FormatInt(telegramID, 10)
	if !strings.HasSuffix(current.Username, suffix) && strings.HasSuffix(candidate.Username, suffix) {
		return candidate
	}
	return current
}
