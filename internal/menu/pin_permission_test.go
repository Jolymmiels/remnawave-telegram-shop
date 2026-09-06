package menu

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-telegram/bot"
)

func TestPinDenialForgetsOnlyConfirmedFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   int
		body   string
		forget bool
	}{
		{"forbidden", 403, "Forbidden: bot is not a member of the chat", true},
		{"rights", 400, "Bad Request: not enough rights to pin a message", true},
		{"admin", 400, "Bad Request: CHAT_ADMIN_REQUIRED", true},
		{"other_bad_request", 400, "Bad Request: unknown failure", false},
		{"server_error", 500, "Internal Server Error", false},
		{"rate_limit", 429, "Too Many Requests", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			b, api := newTelegram(t)
			api.fail = func(apiCall) (int, string) { return tc.code, tc.body }
			store := newStore()
			_ = store.TrackPin(ctx, 123, 42, 50)
			s := newService(t, store, 123, "")
			if err := s.replacePin(ctx, b, 42, 100); err == nil {
				t.Fatal("pin error lost")
			}
			pins, err := store.ListPins(ctx, 123, 42)
			if err != nil {
				t.Fatal(err)
			}
			want := []int{50, 100}
			if tc.forget {
				want = []int{50}
			}
			if !reflect.DeepEqual(pins, want) {
				t.Fatalf("pins=%v want=%v", pins, want)
			}
			if len(api.calls) != 1 || api.calls[0].method != "pinChatMessage" {
				t.Fatal("failed pin removed old menu")
			}
		})
	}
}

func TestRepeatedPinDenialsDoNotAccumulateRows(t *testing.T) {
	ctx := context.Background()
	b, api := newTelegram(t)
	api.fail = func(apiCall) (int, string) { return 403, "Forbidden" }
	store := newStore()
	_ = store.TrackPin(ctx, 123, 42, 50)
	s := newService(t, store, 123, "")
	for id := 100; id < 110; id++ {
		if err := s.replacePin(ctx, b, 42, id); !errors.Is(err, bot.ErrorForbidden) {
			t.Fatalf("original pin error lost: %v", err)
		}
	}
	pins, _ := store.ListPins(ctx, 123, 42)
	if !reflect.DeepEqual(pins, []int{50}) {
		t.Fatalf("uncreated pins retained: %v", pins)
	}
}

func TestPinDenialCleanupFailureKeepsRecoveryStateAndOriginalError(t *testing.T) {
	ctx := context.Background()
	b, api := newTelegram(t)
	api.fail = func(apiCall) (int, string) { return 403, "Forbidden" }
	store := newStore()
	store.failure = "forget"
	s := newService(t, store, 123, "")
	if err := s.replacePin(ctx, b, 42, 100); !errors.Is(err, bot.ErrorForbidden) {
		t.Fatalf("original error lost: %v", err)
	}
	pins, _ := store.ListPins(ctx, 123, 42)
	if !reflect.DeepEqual(pins, []int{100}) {
		t.Fatalf("recovery state lost: %v", pins)
	}
}

func TestUncertainPinErrorsAreNeverClassifiedAsPermissionDenials(t *testing.T) {
	for _, err := range []error{nil, context.Canceled, context.DeadlineExceeded, fmt.Errorf("request timed out: %w", context.DeadlineExceeded), errors.New("not enough rights")} {
		if pinPermissionDenied(err) {
			t.Fatalf("uncertain error classified as permission denial: %v", err)
		}
	}
}
