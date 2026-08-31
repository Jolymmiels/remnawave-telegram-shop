package sync

import (
	"testing"

	"remnawave-tg-shop-bot/internal/remnawave"
)

func TestPreferManagedUser(t *testing.T) {
	const telegramID int64 = 6123456789
	managed := remnawave.User{ID: 20, Username: "42_6123456789", SubscriptionUrl: "managed"}
	unmanaged := remnawave.User{ID: 10, Username: "imported-user", SubscriptionUrl: "unmanaged"}

	for _, tc := range []struct {
		name      string
		current   remnawave.User
		candidate remnawave.User
	}{
		{name: "managed candidate replaces unmanaged current", current: unmanaged, candidate: managed},
		{name: "managed current is retained", current: managed, candidate: unmanaged},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := preferManagedUser(tc.current, tc.candidate, telegramID)
			if got.ID != managed.ID {
				t.Fatalf("expected managed user %d, got %d", managed.ID, got.ID)
			}
		})
	}
}

func TestPreferManagedUserRequiresExactSuffix(t *testing.T) {
	const telegramID int64 = 12345
	current := remnawave.User{ID: 10, Username: "existing"}
	candidate := remnawave.User{ID: 20, Username: "42_12345_backup"}

	got := preferManagedUser(current, candidate, telegramID)
	if got.ID != current.ID {
		t.Fatalf("username containing but not ending in Telegram ID must not be preferred: got %d", got.ID)
	}
}
