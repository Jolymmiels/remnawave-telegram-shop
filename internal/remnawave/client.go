package remnawave

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/internal/database"
	"remnawave-tg-shop-bot/utils"
	"strconv"
	"strings"
	"time"

	remapi "github.com/Jolymmiles/remnawave-api-go/v2/api"
	"github.com/google/uuid"
)

type Client struct {
	client *remapi.ClientExt
}

type headerTransport struct {
	base    http.RoundTripper
	local   bool
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())

	if t.local {
		r.Header.Set("x-forwarded-for", "127.0.0.1")
		r.Header.Set("x-forwarded-proto", "https")
	}

	for key, value := range t.headers {
		r.Header.Set(key, value)
	}

	return t.base.RoundTrip(r)
}

func NewClient(baseURL, token, mode string) *Client {
	local := mode == "local"
	headers := config.RemnawaveHeaders()

	client := &http.Client{
		Transport: &headerTransport{
			base:    http.DefaultTransport,
			local:   local,
			headers: headers,
		},
	}

	api, err := remapi.NewClient(baseURL, remapi.StaticToken{Token: token}, remapi.WithClient(client))
	if err != nil {
		panic(err)
	}
	return &Client{client: remapi.NewClientExt(api)}
}

func (r *Client) Ping(ctx context.Context) error {
	_, err := r.client.Users().GetAllUsers(ctx, 1, 0)
	return err
}

func (r *Client) GetUsers(ctx context.Context) (*[]remapi.UserItemInfo, error) {
	pager := remapi.NewPaginationHelper(250)
	users := make([]remapi.UserItemInfo, 0)

	for {
		resp, err := r.client.Users().GetAllUsers(ctx, pager.Limit, pager.Offset)
		if err != nil {
			return nil, err
		}

		response := resp.(*remapi.GetAllUsersResponse).GetResponse()
		users = append(users, response.Users...)

		if len(response.Users) < pager.Limit {
			break
		}

		if !pager.NextPage() {
			break
		}
	}

	return &users, nil
}

func (r *Client) DecreaseSubscription(ctx context.Context, telegramId int64, trafficLimit int, days int) (*time.Time, error) {

	resp, err := r.client.Users().GetUserByTelegramId(ctx, strconv.FormatInt(telegramId, 10))
	if err != nil {
		return nil, err
	}

	usersResp, ok := resp.(*remapi.UsersResponse)
	if !ok {
		return nil, errors.New("unknown response type")
	}

	users := usersResp.GetResponse()
	if len(users) == 0 {
		return nil, fmt.Errorf("user with telegramId %d not found", telegramId)
	}

	var existingUser *remapi.UserItemInfo
	suffix := fmt.Sprintf("_%d", telegramId)

	for i := range users {
		if strings.Contains(users[i].Username, suffix) {
			existingUser = &users[i]
			break
		}
	}

	if existingUser == nil {
		existingUser = &users[0]
	}

	updated, err := r.updateUser(ctx, existingUser, trafficLimit, days)
	if err != nil {
		return nil, err
	}

	return &updated.ExpireAt, nil
}

func (r *Client) CreateOrUpdateUser(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int, isTrialUser bool) (*remapi.UserItemInfo, error) {
	resp, err := r.client.Users().GetUserByTelegramId(ctx, strconv.FormatInt(telegramId, 10))
	if err != nil {
		return nil, err
	}

	usersResp, ok := resp.(*remapi.UsersResponse)
	if !ok {
		return nil, errors.New("unknown response type")
	}

	users := usersResp.GetResponse()
	if len(users) == 0 {
		return r.createUser(ctx, customerId, telegramId, trafficLimit, days, isTrialUser)
	}

	var existingUser *remapi.UserItemInfo
	suffix := fmt.Sprintf("_%d", telegramId)

	for i := range users {
		if strings.Contains(users[i].Username, suffix) {
			existingUser = &users[i]
			break
		}
	}

	if existingUser == nil {
		existingUser = &users[0]
	}

	return r.updateUser(ctx, existingUser, trafficLimit, days)
}

func (r *Client) CreateOrUpdateCustomer(
	ctx context.Context,
	customer *database.Customer,
	trafficLimit int,
	days int,
	isTrialUser bool,
) (*remapi.UserItemInfo, error) {
	if customer == nil {
		return nil, errors.New("customer is nil")
	}

	if customer.RemnawaveUserUUID != nil && *customer.RemnawaveUserUUID != "" {
		resp, err := r.client.Users().GetUserByUuid(ctx, *customer.RemnawaveUserUUID)
		if err == nil {
			if userResp, ok := resp.(*remapi.UserResponse); ok {
				user := userResp.GetResponse()
				return r.updateUser(ctx, &user, trafficLimit, days)
			}
		}
	}

	if customer.TelegramID != 0 {
		return r.CreateOrUpdateUser(ctx, customer.ID, customer.TelegramID, trafficLimit, days, isTrialUser)
	}

	return r.createUser(ctx, customer.ID, 0, trafficLimit, days, isTrialUser)
}

func (r *Client) updateUser(ctx context.Context, existingUser *remapi.UserItemInfo, trafficLimit int, days int) (*remapi.UserItemInfo, error) {

	newExpire := getNewExpire(days, existingUser.ExpireAt)

	resp, err := r.client.InternalSquad().GetInternalSquads(ctx)
	if err != nil {
		return nil, err
	}

	squads := resp.(*remapi.InternalSquadsResponse).GetResponse()

	selectedSquads := config.SquadUUIDs()

	squadId := make([]uuid.UUID, 0, len(selectedSquads))
	for _, squad := range squads.GetInternalSquads() {
		if len(selectedSquads) > 0 {
			if _, isExist := selectedSquads[squad.UUID]; !isExist {
				continue
			} else {
				squadId = append(squadId, squad.UUID)
			}
		} else {
			squadId = append(squadId, squad.UUID)
		}
	}

	userUpdate := &remapi.UpdateUserRequest{
		UUID:                 remapi.NewOptUUID(existingUser.UUID),
		ExpireAt:             remapi.NewOptDateTime(newExpire),
		Status:               remapi.NewOptUpdateUserRequestStatus(remapi.UpdateUserRequestStatusACTIVE),
		TrafficLimitBytes:    remapi.NewOptInt(trafficLimit),
		ActiveInternalSquads: squadId,
		TrafficLimitStrategy: remapi.NewOptUpdateUserRequestTrafficLimitStrategy(getUpdateStrategy(config.TrafficLimitResetStrategy())),
	}

	externalSquad := config.ExternalSquadUUID()
	if externalSquad != uuid.Nil {
		userUpdate.ExternalSquadUuid = remapi.NewOptNilUUID(externalSquad)
	}

	tag := config.RemnawaveTag()
	if tag != "" {
		userUpdate.Tag = remapi.NewOptNilString(tag)
	}

	var username string
	if ctx.Value(utils.ContextKeyUsername) != nil {
		username = ctx.Value(utils.ContextKeyUsername).(string)
		userUpdate.Description = remapi.NewOptNilString(username)
	} else {
		username = ""
	}

	updateUser, err := r.client.Users().UpdateUser(ctx, userUpdate)
	if err != nil {
		return nil, err
	}
	if value, ok := updateUser.(*remapi.InternalServerError); ok {
		return nil, errors.New("error while updating user. message: " + value.GetMessage().Value + ". code: " + value.GetErrorCode().Value)
	}

	tgid, _ := existingUser.TelegramId.Get()
	slog.Info("updated user", "telegramId", utils.MaskHalf(strconv.Itoa(tgid)), "username", utils.MaskHalf(username), "days", days)
	result := updateUser.(*remapi.UserResponse).GetResponse()
	return &result, nil
}

func (r *Client) createUser(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int, isTrialUser bool) (*remapi.UserItemInfo, error) {
	expireAt := time.Now().UTC().AddDate(0, 0, days)
	username := generateUsername(customerId, telegramId)

	resp, err := r.client.InternalSquad().GetInternalSquads(ctx)
	if err != nil {
		return nil, err
	}

	squads := resp.(*remapi.InternalSquadsResponse).GetResponse()

	selectedSquads := config.SquadUUIDs()
	if isTrialUser {
		selectedSquads = config.TrialInternalSquads()
	}

	squadId := make([]uuid.UUID, 0, len(selectedSquads))
	for _, squad := range squads.GetInternalSquads() {
		if len(selectedSquads) > 0 {
			if _, isExist := selectedSquads[squad.UUID]; !isExist {
				continue
			} else {
				squadId = append(squadId, squad.UUID)
			}
		} else {
			squadId = append(squadId, squad.UUID)
		}
	}

	externalSquad := config.ExternalSquadUUID()
	if isTrialUser {
		externalSquad = config.TrialExternalSquadUUID()
	}

	strategy := config.TrafficLimitResetStrategy()
	if isTrialUser {
		strategy = config.TrialTrafficLimitResetStrategy()
	}

	createUserRequest := remapi.CreateUserRequest{
		Username:             username,
		ActiveInternalSquads: squadId,
		Status:               remapi.NewOptCreateUserRequestStatus(remapi.CreateUserRequestStatusACTIVE),
		ExpireAt:             expireAt,
		TrafficLimitStrategy: remapi.NewOptCreateUserRequestTrafficLimitStrategy(getCreateStrategy(strategy)),
		TrafficLimitBytes:    remapi.NewOptInt(trafficLimit),
	}
	if telegramId != 0 {
		createUserRequest.TelegramId = remapi.NewOptNilInt(int(telegramId))
	}
	if externalSquad != uuid.Nil {
		createUserRequest.ExternalSquadUuid = remapi.NewOptNilUUID(externalSquad)
	}
	tag := config.RemnawaveTag()
	if isTrialUser {
		tag = config.TrialRemnawaveTag()
	}
	if tag != "" {
		createUserRequest.Tag = remapi.NewOptNilString(tag)
	}

	var tgUsername string
	if ctx.Value(utils.ContextKeyUsername) != nil {
		tgUsername = ctx.Value(utils.ContextKeyUsername).(string)
		createUserRequest.Description = remapi.NewOptString(ctx.Value(utils.ContextKeyUsername).(string))
	} else {
		tgUsername = ""
	}

	userCreate, err := r.client.Users().CreateUser(ctx, &createUserRequest)
	if err != nil {
		return nil, err
	}
	slog.Info("created user", "telegramId", utils.MaskHalf(strconv.FormatInt(telegramId, 10)), "username", utils.MaskHalf(tgUsername), "days", days)
	result := userCreate.(*remapi.UserResponse).GetResponse()
	return &result, nil
}

func generateUsername(customerId int64, telegramId int64) string {
	return fmt.Sprintf("%d_%d", customerId, telegramId)
}

func getNewExpire(daysToAdd int, currentExpire time.Time) time.Time {
	if daysToAdd <= 0 {
		if currentExpire.AddDate(0, 0, daysToAdd).Before(time.Now()) {
			return time.Now().UTC().AddDate(0, 0, 1)
		} else {
			return currentExpire.AddDate(0, 0, daysToAdd)
		}
	}

	if currentExpire.Before(time.Now().UTC()) || currentExpire.IsZero() {
		return time.Now().UTC().AddDate(0, 0, daysToAdd)
	}

	return currentExpire.AddDate(0, 0, daysToAdd)
}

func getCreateStrategy(s string) remapi.CreateUserRequestTrafficLimitStrategy {
	switch s {
	case "DAY":
		return remapi.CreateUserRequestTrafficLimitStrategyDAY
	case "WEEK":
		return remapi.CreateUserRequestTrafficLimitStrategyWEEK
	case "NO_RESET":
		return remapi.CreateUserRequestTrafficLimitStrategyNORESET
	default:
		return remapi.CreateUserRequestTrafficLimitStrategyMONTH
	}
}

func getUpdateStrategy(s string) remapi.UpdateUserRequestTrafficLimitStrategy {
	switch s {
	case "DAY":
		return remapi.UpdateUserRequestTrafficLimitStrategyDAY
	case "WEEK":
		return remapi.UpdateUserRequestTrafficLimitStrategyWEEK
	case "NO_RESET":
		return remapi.UpdateUserRequestTrafficLimitStrategyNORESET
	default:
		return remapi.UpdateUserRequestTrafficLimitStrategyMONTH
	}
}
