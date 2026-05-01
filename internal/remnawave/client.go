package remnawave

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/utils"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when the API responds with 404.
var ErrNotFound = errors.New("not found")

// ctxKey is an unexported type for context keys in this package.
type ctxKey string

// CtxKeyUsername is the context key used to pass the Telegram username.
const CtxKeyUsername ctxKey = "username"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type headerTransport struct {
	base       http.RoundTripper
	headers    map[string]string
	forceLocal bool
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())

	host := r.URL.Hostname()
	if t.forceLocal || strings.HasPrefix(host, "remnawave") || host == "127.0.0.1" || host == "localhost" {
		r.Header.Set("x-forwarded-for", "127.0.0.1")
		r.Header.Set("x-forwarded-proto", "https")
	}

	for key, value := range t.headers {
		r.Header.Set(key, value)
	}

	return t.base.RoundTrip(r)
}

func NewClient(baseURL, token, mode string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	headers := config.RemnawaveHeaders()
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Authorization"] = "Bearer " + token
	forceLocal := mode == "local"

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &headerTransport{
			base:       http.DefaultTransport,
			headers:    headers,
			forceLocal: forceLocal,
		},
	}

	return &Client{
		httpClient: client,
		baseURL:    baseURL,
	}
}

// ---------------------------------------------------------------------------
// Generic HTTP helpers
// ---------------------------------------------------------------------------

func (r *Client) doRequest(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return respBody, resp.StatusCode, ErrNotFound
	}

	if resp.StatusCode >= 400 {
		var apiErr apiErrorResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
			return respBody, resp.StatusCode, fmt.Errorf("API error %d: %s (code: %s)", resp.StatusCode, apiErr.Message, apiErr.ErrorCode)
		}
		return respBody, resp.StatusCode, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

func (r *Client) doJSON(ctx context.Context, method, path string, body, result any) error {
	respBody, _, err := r.doRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Ping
// ---------------------------------------------------------------------------

func (r *Client) Ping(ctx context.Context) error {
	path := fmt.Sprintf("/api/users?size=%d&start=%d", 1, 0)
	return r.doJSON(ctx, http.MethodGet, path, nil, nil)
}

// ---------------------------------------------------------------------------
// Users — list
// ---------------------------------------------------------------------------

func (r *Client) GetUsers(ctx context.Context) ([]User, error) {
	const pageSize = 250
	var users []User

	for offset := 0; ; offset += pageSize {
		path := fmt.Sprintf("/api/users?size=%d&start=%d", pageSize, offset)
		var page getAllUsersResponse
		if err := r.doJSON(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, fmt.Errorf("fetch users at offset %d: %w", offset, err)
		}

		users = append(users, page.Response.Users...)

		if len(page.Response.Users) < pageSize {
			break
		}
	}

	return users, nil
}

// ---------------------------------------------------------------------------
// Users — get by Telegram ID
// ---------------------------------------------------------------------------

func (r *Client) getUsersByTelegramID(ctx context.Context, telegramID int64) ([]User, error) {
	var resp apiResponse[[]User]
	err := r.doJSON(ctx, http.MethodGet, "/api/users/by-telegram-id/"+strconv.FormatInt(telegramID, 10), nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Response, nil
}

// ---------------------------------------------------------------------------
// Internal squads
// ---------------------------------------------------------------------------

func (r *Client) getInternalSquads(ctx context.Context) ([]internalSquadItem, error) {
	var resp apiResponse[internalSquadsResponse]
	if err := r.doJSON(ctx, http.MethodGet, "/api/internal-squads", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Response.InternalSquads, nil
}

func filterSquadsBySelection(allSquads []internalSquadItem, selected map[uuid.UUID]uuid.UUID) []uuid.UUID {
	if len(selected) == 0 {
		result := make([]uuid.UUID, 0, len(allSquads))
		for _, s := range allSquads {
			result = append(result, s.UUID)
		}
		return result
	}
	result := make([]uuid.UUID, 0, len(selected))
	for _, s := range allSquads {
		if _, ok := selected[s.UUID]; ok {
			result = append(result, s.UUID)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// DecreaseSubscription
// ---------------------------------------------------------------------------

func (r *Client) DecreaseSubscription(ctx context.Context, telegramId int64, trafficLimit int, days int) (*time.Time, error) {
	users, err := r.getUsersByTelegramID(ctx, telegramId)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user with telegramId %d not found", telegramId)
	}

	existingUser := findUserBySuffix(users, telegramId)

	updated, err := r.updateUser(ctx, existingUser, trafficLimit, days)
	if err != nil {
		return nil, err
	}

	return &updated.ExpireAt, nil
}

// ---------------------------------------------------------------------------
// CreateOrUpdateUser
// ---------------------------------------------------------------------------

func (r *Client) CreateOrUpdateUser(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int, isTrialUser bool) (*User, error) {
	users, err := r.getUsersByTelegramID(ctx, telegramId)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return r.createUser(ctx, customerId, telegramId, trafficLimit, days, isTrialUser)
	}

	existingUser := findUserBySuffix(users, telegramId)
	return r.updateUser(ctx, existingUser, trafficLimit, days)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func findUserBySuffix(users []User, telegramId int64) *User {
	suffix := fmt.Sprintf("_%d", telegramId)
	for i := range users {
		if strings.Contains(users[i].Username, suffix) {
			return &users[i]
		}
	}
	return &users[0]
}

// UsernameFromCtx extracts the Telegram username stored under CtxKeyUsername.
// Returns empty string if the value is missing or not a string.
func UsernameFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(CtxKeyUsername).(string); ok {
		return v
	}
	return ""
}

func (r *Client) updateUser(ctx context.Context, existingUser *User, trafficLimit int, days int) (*User, error) {
	newExpire := getNewExpire(days, existingUser.ExpireAt)

	squads, err := r.getInternalSquads(ctx)
	if err != nil {
		return nil, err
	}

	selectedSquads := config.SquadUUIDs()
	squadIds := filterSquadsBySelection(squads, selectedSquads)

	userUpdate := &UpdateUserRequest{
		UUID:                 &existingUser.UUID,
		ExpireAt:             &newExpire,
		Status:               "ACTIVE",
		TrafficLimitBytes:    &trafficLimit,
		ActiveInternalSquads: squadIds,
		TrafficLimitStrategy: normalizeStrategy(config.TrafficLimitResetStrategy()),
	}

	externalSquad := config.ExternalSquadUUID()
	if externalSquad != uuid.Nil {
		userUpdate.ExternalSquadUuid = &externalSquad
	}

	tag := config.RemnawaveTag()
	if tag != "" {
		userUpdate.Tag = &tag
	}

	username := UsernameFromCtx(ctx)
	if username != "" {
		userUpdate.Description = &username
	}

	var resp apiResponse[User]
	if err := r.doJSON(ctx, http.MethodPatch, "/api/users", userUpdate, &resp); err != nil {
		return nil, err
	}

	tgid := ""
	if existingUser.TelegramID != nil {
		tgid = strconv.FormatInt(*existingUser.TelegramID, 10)
	}
	slog.Info("updated user", "telegramId", utils.MaskHalf(tgid), "username", utils.MaskHalf(username), "days", days)
	return &resp.Response, nil
}

func (r *Client) createUser(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int, isTrialUser bool) (*User, error) {
	expireAt := time.Now().UTC().AddDate(0, 0, days)
	username := generateUsername(customerId, telegramId)

	squads, err := r.getInternalSquads(ctx)
	if err != nil {
		return nil, err
	}

	selectedSquads := config.SquadUUIDs()
	if isTrialUser {
		selectedSquads = config.TrialInternalSquads()
	}
	squadIds := filterSquadsBySelection(squads, selectedSquads)

	externalSquad := config.ExternalSquadUUID()
	if isTrialUser {
		externalSquad = config.TrialExternalSquadUUID()
	}

	strategy := config.TrafficLimitResetStrategy()
	if isTrialUser {
		strategy = config.TrialTrafficLimitResetStrategy()
	}

	tid := int(telegramId)
	createReq := &CreateUserRequest{
		Username:             username,
		ActiveInternalSquads: squadIds,
		Status:               "ACTIVE",
		TelegramID:           &tid,
		ExpireAt:             expireAt,
		TrafficLimitStrategy: normalizeStrategy(strategy),
		TrafficLimitBytes:    &trafficLimit,
	}
	if externalSquad != uuid.Nil {
		createReq.ExternalSquadUuid = &externalSquad
	}
	tag := config.RemnawaveTag()
	if isTrialUser {
		tag = config.TrialRemnawaveTag()
	}
	if tag != "" {
		createReq.Tag = &tag
	}

	tgUsername := UsernameFromCtx(ctx)
	if tgUsername != "" {
		createReq.Description = &tgUsername
	}

	var resp apiResponse[User]
	if err := r.doJSON(ctx, http.MethodPost, "/api/users", createReq, &resp); err != nil {
		return nil, err
	}
	slog.Info("created user", "telegramId", utils.MaskHalf(strconv.FormatInt(telegramId, 10)), "username", utils.MaskHalf(tgUsername), "days", days)
	return &resp.Response, nil
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

func generateUsername(customerId int64, telegramId int64) string {
	return fmt.Sprintf("%d_%d", customerId, telegramId)
}

func getNewExpire(daysToAdd int, currentExpire time.Time) time.Time {
	if daysToAdd <= 0 {
		if currentExpire.AddDate(0, 0, daysToAdd).Before(time.Now()) {
			return time.Now().UTC().AddDate(0, 0, 1)
		}
		return currentExpire.AddDate(0, 0, daysToAdd)
	}

	if currentExpire.Before(time.Now().UTC()) || currentExpire.IsZero() {
		return time.Now().UTC().AddDate(0, 0, daysToAdd)
	}

	return currentExpire.AddDate(0, 0, daysToAdd)
}

func normalizeStrategy(s string) string {
	upper := strings.ToUpper(s)
	switch upper {
	case "DAY", "WEEK", "NO_RESET", "MONTH":
		return upper
	default:
		return "MONTH"
	}
}
