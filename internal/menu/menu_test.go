package menu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type memoryStore struct {
	mu      sync.Mutex
	photos  map[string]PhotoRecord
	pins    map[string]map[int]bool
	failure string
}

func newStore() *memoryStore {
	return &memoryStore{photos: map[string]PhotoRecord{}, pins: map[string]map[int]bool{}}
}

func (s *memoryStore) LoadPhoto(_ context.Context, botID int64, path string) (PhotoRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure == "load" {
		return PhotoRecord{}, errors.New("load failed")
	}
	return s.photos[fmt.Sprint(botID)+":"+path], nil
}
func (s *memoryStore) SavePhoto(_ context.Context, botID int64, path string, record PhotoRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure == "save" {
		return errors.New("save failed")
	}
	s.photos[fmt.Sprint(botID)+":"+path] = record
	return nil
}
func (s *memoryStore) ListPins(_ context.Context, botID, chatID int64) ([]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure == "list" {
		return nil, errors.New("list failed")
	}
	var pins []int
	for id := range s.pins[fmt.Sprintf("%d:%d", botID, chatID)] {
		pins = append(pins, id)
	}
	sort.Ints(pins)
	return pins, nil
}
func (s *memoryStore) TrackPin(_ context.Context, botID, chatID int64, messageID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure == "track" {
		return errors.New("track failed")
	}
	key := fmt.Sprintf("%d:%d", botID, chatID)
	if s.pins[key] == nil {
		s.pins[key] = map[int]bool{}
	}
	s.pins[key][messageID] = true
	return nil
}
func (s *memoryStore) ForgetPin(_ context.Context, botID, chatID int64, messageID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failure == "forget" {
		return errors.New("forget failed")
	}
	delete(s.pins[fmt.Sprintf("%d:%d", botID, chatID)], messageID)
	return nil
}

type apiCall struct {
	method string
	values url.Values
	upload bool
}
type fakeTelegram struct {
	mu     sync.Mutex
	calls  []apiCall
	fail   func(apiCall) (int, string)
	nextID int
}

func newTelegram(t *testing.T) (*bot.Bot, *fakeTelegram) {
	t.Helper()
	api := &fakeTelegram{nextID: 100}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(12 << 20); err != nil {
			t.Error(err)
			w.WriteHeader(500)
			return
		}
		defer r.MultipartForm.RemoveAll()
		api.mu.Lock()
		defer api.mu.Unlock()
		call := apiCall{method: filepath.Base(r.URL.Path), values: r.Form, upload: len(r.MultipartForm.File) > 0}
		api.calls = append(api.calls, call)
		w.Header().Set("Content-Type", "application/json")
		if api.fail != nil {
			if code, description := api.fail(call); code != 0 {
				w.WriteHeader(code)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error_code": code, "description": description})
				return
			}
		}
		var result any = true
		if strings.HasPrefix(call.method, "send") || strings.HasPrefix(call.method, "edit") {
			id, _ := strconv.Atoi(call.values.Get("message_id"))
			if id == 0 {
				api.nextID++
				id = api.nextID
			}
			chatID, _ := strconv.ParseInt(call.values.Get("chat_id"), 10, 64)
			msg := models.Message{ID: id, Chat: models.Chat{ID: chatID}}
			if call.method == "sendPhoto" {
				msg.Photo = []models.PhotoSize{{FileID: "small"}, {FileID: "cached-photo"}}
			}
			result = msg
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
	}))
	t.Cleanup(server.Close)
	b, err := bot.New("123:test", bot.WithSkipGetMe(), bot.WithServerURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	return b, api
}

func photoFile(t *testing.T, path string, value uint8) string {
	t.Helper()
	if path == "" {
		path = filepath.Join(t.TempDir(), "menu.png")
	}
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	img.Set(0, 0, color.RGBA{R: value, A: 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
func newService(t *testing.T, store Store, botID int64, path string) *Service {
	t.Helper()
	s, err := New(context.Background(), store, botID, path)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func send(t *testing.T, s *Service, b *bot.Bot) {
	t.Helper()
	if _, err := s.Send(context.Background(), b, 42, "<b>Menu</b>", models.InlineKeyboardMarkup{}); err != nil {
		t.Fatal(err)
	}
}

func TestPhotoCacheReuseRestartReplacementAndBotIsolation(t *testing.T) {
	b, api := newTelegram(t)
	store := newStore()
	path := photoFile(t, "", 1)
	s := newService(t, store, 123, path)
	send(t, s, b)
	send(t, s, b)
	send(t, newService(t, store, 123, path), b)
	send(t, newService(t, store, 456, path), b)
	photoFile(t, path, 2)
	send(t, newService(t, store, 123, path), b)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	send(t, newService(t, store, 123, path), b)
	wantUploads := []bool{true, false, false, true, true, false}
	for i, call := range api.calls {
		if call.method != "sendPhoto" || call.upload != wantUploads[i] {
			t.Fatalf("call %d: %+v", i, call)
		}
		if !call.upload && call.values.Get("photo") != "cached-photo" {
			t.Fatalf("file_id was not reused: %+v", call)
		}
		if call.values.Get("caption") != "<b>Menu</b>" || call.values.Get("parse_mode") != "HTML" || call.values.Get("reply_markup") == "" {
			t.Fatalf("lost caption/keyboard: %+v", call)
		}
	}
}

func TestPhotoValidation(t *testing.T) {
	store := newStore()
	path := filepath.Join(t.TempDir(), "photo.png")
	if _, err := New(context.Background(), store, 1, path); err == nil {
		t.Fatal("missing file accepted")
	}
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(context.Background(), store, 1, path); err == nil {
		t.Fatal("invalid image accepted")
	}
	if err := os.WriteFile(path, make([]byte, 10*1024*1024+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(context.Background(), store, 1, path); err == nil {
		t.Fatal("oversized image accepted")
	}
	store.failure = "load"
	if _, err := New(context.Background(), store, 1, path); err == nil {
		t.Fatal("cache load failure ignored")
	}
}

func TestRejectedFileIDReuploadsButOtherErrorsDoNot(t *testing.T) {
	for _, tc := range []struct {
		description string
		code        int
		retries     bool
	}{
		{"Bad Request: wrong file identifier/HTTP URL specified", 400, true},
		{"Bad Request: FILE_REFERENCE_EXPIRED", 400, true},
		{"Bad Request: message caption is too long", 400, false},
		{"Bad Request: can't parse entities", 400, false},
		{"Forbidden: bot was blocked by the user", 403, false},
		{"Internal Server Error", 500, false},
	} {
		t.Run(tc.description, func(t *testing.T) {
			b, api := newTelegram(t)
			s := newService(t, newStore(), 123, photoFile(t, "", 1))
			send(t, s, b)
			api.fail = func(c apiCall) (int, string) {
				if !c.upload {
					return tc.code, tc.description
				}
				return 0, ""
			}
			_, err := s.Send(context.Background(), b, 42, "Menu", nil)
			if (err == nil) != tc.retries {
				t.Fatalf("unexpected result: %v", err)
			}
			want := 2
			if tc.retries {
				want = 3
			}
			if len(api.calls) != want {
				t.Fatalf("got %d API calls, want %d", len(api.calls), want)
			}
		})
	}
}

func TestPhotoCacheWriteRetriesWithoutUploadingAgain(t *testing.T) {
	b, api := newTelegram(t)
	store := newStore()
	s := newService(t, store, 123, photoFile(t, "", 1))
	store.failure = "save"
	send(t, s, b)
	store.failure = ""
	send(t, s, b)
	record, err := store.LoadPhoto(context.Background(), 123, s.path)
	if err != nil || record.FileID != "cached-photo" || api.calls[1].upload {
		t.Fatalf("cache retry failed: %+v %v", record, err)
	}
}

func TestConcurrentPhotoRequestsUploadOnce(t *testing.T) {
	b, api := newTelegram(t)
	s := newService(t, newStore(), 123, photoFile(t, "", 1))
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Send(context.Background(), b, 42, "Menu", nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	uploads := 0
	for _, call := range api.calls {
		if call.upload {
			uploads++
		}
	}
	if uploads != 1 || len(api.calls) != 12 {
		t.Fatalf("uploads=%d calls=%d", uploads, len(api.calls))
	}
}

func TestStartReplacesOnlyTrackedPinsAcrossRestart(t *testing.T) {
	for _, photo := range []bool{false, true} {
		t.Run(fmt.Sprint(photo), func(t *testing.T) {
			b, api := newTelegram(t)
			store := newStore()
			path := ""
			if photo {
				path = photoFile(t, "", 1)
			}
			ctx := context.Background()
			first, err := newService(t, store, 123, path).SendStart(ctx, b, 42, "Menu", nil)
			if err != nil {
				t.Fatal(err)
			}
			// A different bot/chat must never be unpinned.
			_ = store.TrackPin(ctx, 456, 42, 9)
			_ = store.TrackPin(ctx, 123, 43, 10)
			second, err := newService(t, store, 123, path).SendStart(ctx, b, 42, "Menu", nil)
			if err != nil {
				t.Fatal(err)
			}
			method := "sendMessage"
			if photo {
				method = "sendPhoto"
			}
			want := []string{method, "pinChatMessage", method, "pinChatMessage", "unpinChatMessage"}
			for i, call := range api.calls {
				if call.method != want[i] {
					t.Fatalf("call %d: %+v", i, call)
				}
				if call.method == "pinChatMessage" && call.values.Get("disable_notification") != "true" {
					t.Fatal("noisy pin")
				}
			}
			if api.calls[4].values.Get("message_id") != strconv.Itoa(first.ID) {
				t.Fatal("wrong pin removed")
			}
			pins, _ := store.ListPins(ctx, 123, 42)
			if !reflect.DeepEqual(pins, []int{second.ID}) {
				t.Fatalf("pins: %v", pins)
			}
		})
	}
}

func TestPinAndPersistenceFailuresKeepOldMenuTracked(t *testing.T) {
	for _, failure := range []string{"send", "pin", "list", "track", "unpin", "forget", "deleted"} {
		t.Run(failure, func(t *testing.T) {
			ctx := context.Background()
			b, api := newTelegram(t)
			store := newStore()
			_ = store.TrackPin(ctx, 123, 42, 50)
			s := newService(t, store, 123, "")
			store.failure = failure
			api.fail = func(c apiCall) (int, string) {
				if failure == "send" && c.method == "sendMessage" {
					return 500, "unavailable"
				}
				if failure == "pin" && c.method == "pinChatMessage" {
					return 400, "not enough rights"
				}
				if failure == "unpin" && c.method == "unpinChatMessage" {
					return 500, "unavailable"
				}
				if failure == "deleted" && c.method == "unpinChatMessage" {
					return 400, "Bad Request: message to unpin not found"
				}
				return 0, ""
			}
			_, err := s.SendStart(ctx, b, 42, "Menu", nil)
			if (err != nil) != (failure == "send") {
				t.Fatalf("menu delivery error: %v", err)
			}
			if failure == "pin" || failure == "list" || failure == "track" || failure == "send" {
				for _, call := range api.calls {
					if call.method == "unpinChatMessage" {
						t.Fatal("old pin removed before new pin succeeds")
					}
				}
			}
			store.failure = ""
			pins, _ := store.ListPins(ctx, 123, 42)
			containsOld := false
			for _, id := range pins {
				if id == 50 {
					containsOld = true
				}
			}
			if containsOld != (failure != "deleted") {
				t.Fatalf("lost cleanup state: %v", pins)
			}
			api.fail = nil
			message, err := newService(t, store, 123, "").SendStart(ctx, b, 42, "Menu", nil)
			if err != nil {
				t.Fatal(err)
			}
			pins, _ = store.ListPins(ctx, 123, 42)
			if !reflect.DeepEqual(pins, []int{message.ID}) {
				t.Fatalf("cleanup not retried: %v", pins)
			}
		})
	}
}

func TestConcurrentStartsLeaveOnePin(t *testing.T) {
	b, api := newTelegram(t)
	store := newStore()
	s := newService(t, store, 123, "")
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.SendStart(context.Background(), b, 42, "Menu", nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	pins, _ := store.ListPins(context.Background(), 123, 42)
	if !reflect.DeepEqual(pins, []int{api.nextID}) {
		t.Fatalf("pins: %v", pins)
	}
}

func TestEditPreservesPhotoAndLegacyTextMenus(t *testing.T) {
	for _, photo := range []bool{false, true} {
		for _, description := range []string{"", "Bad Request: message is not modified", "Bad Request: message caption is too long"} {
			t.Run(fmt.Sprintf("%t/%s", photo, description), func(t *testing.T) {
				b, api := newTelegram(t)
				message := &models.Message{ID: 50, Chat: models.Chat{ID: 42}}
				if photo {
					message.Photo = []models.PhotoSize{{FileID: "keep-me"}}
				}
				if description != "" {
					api.fail = func(apiCall) (int, string) { return 400, description }
				}
				markup := models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{Text: "Back", CallbackData: "start", Style: "primary", IconCustomEmojiID: "1234"}}}}
				_, err := Edit(context.Background(), b, message, &bot.EditMessageTextParams{ChatID: 42, MessageID: 50, Text: "<b>Section</b>", ParseMode: models.ParseModeHTML, ReplyMarkup: markup})
				if (err != nil) != strings.Contains(description, "too long") {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(api.calls) != 1 {
					t.Fatal("edit sent/deleted another message")
				}
				call := api.calls[0]
				method, field := "editMessageText", "text"
				if photo {
					method, field = "editMessageCaption", "caption"
				}
				if call.method != method || call.values.Get(field) != "<b>Section</b>" || call.values.Get("message_id") != "50" {
					t.Fatalf("bad edit: %+v", call)
				}
				var got models.InlineKeyboardMarkup
				if err := json.Unmarshal([]byte(call.values.Get("reply_markup")), &got); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, markup) {
					t.Fatal("lost inline keyboard styling")
				}
			})
		}
	}
}
