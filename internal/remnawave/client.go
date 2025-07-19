package remnawave

import (
	"context"
	"errors"
	"fmt"
	remapi "github.com/Jolymmiles/remnawave-api-go/api"
	"github.com/google/uuid"
	"log/slog"
	"net/http"
	"remnawave-tg-shop-bot/internal/config"
	"remnawave-tg-shop-bot/utils"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	client *remapi.Client
}

type headerTransport struct {
	base    http.RoundTripper
	xApiKey string
	local   bool
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())

	if t.xApiKey != "" {
		r.Header.Set("X-Api-Key", t.xApiKey)
	}

	if t.local {
		r.Header.Set("x-forwarded-for", "127.0.0.1")
		r.Header.Set("x-forwarded-proto", "https")
	}

	return t.base.RoundTrip(r)
}

func NewClient(baseURL, token, mode string) *Client {
	xApiKey := config.GetXApiKey()
	local := mode == "local"

	client := &http.Client{
		Transport: &headerTransport{
			base:    http.DefaultTransport,
			xApiKey: xApiKey,
			local:   local,
		},
	}

	api, err := remapi.NewClient(baseURL, remapi.StaticToken{Token: token}, remapi.WithClient(client))
	if err != nil {
		panic(err)
	}
	return &Client{client: api}
}

func (r *Client) Ping(ctx context.Context) error {
	params := remapi.UsersControllerGetAllUsersParams{
		Size:  remapi.NewOptFloat64(1),
		Start: remapi.NewOptFloat64(0),
	}
	_, err := r.client.UsersControllerGetAllUsers(ctx, params)
	return err
}

func (r *Client) GetUsers(ctx context.Context) (*[]remapi.UserDto, error) {
	pageSize := float64(250)
	start := float64(0)

	users := make([]remapi.UserDto, 0)
	for {
		resp, err := r.client.UsersControllerGetAllUsers(ctx,
			remapi.UsersControllerGetAllUsersParams{Size: remapi.NewOptFloat64(pageSize), Start: remapi.NewOptFloat64(start)})

		if err != nil {
			return nil, err
		}
		response := resp.GetResponse()

		usersResponse := &response.Users

		users = append(users, *usersResponse...)

		start += float64(len(*usersResponse))

		if start >= response.GetTotal() {
			break
		}
	}

	return &users, nil
}

func (r *Client) CreateOrUpdateUser(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int) (*remapi.UserDto, error) {
	resp, err := r.client.UsersControllerGetUserByTelegramId(ctx, remapi.UsersControllerGetUserByTelegramIdParams{TelegramId: strconv.FormatInt(telegramId, 10)})
	if err != nil {
		return nil, err
	}

	switch v := resp.(type) {

	case *remapi.UsersControllerGetUserByTelegramIdNotFound:
		return r.createUser(ctx, customerId, telegramId, trafficLimit, days)
	case *remapi.UsersDto:
		var existingUser *remapi.UserDto
		for _, panelUser := range v.GetResponse() {
			if strings.Contains(panelUser.Username, fmt.Sprintf("_%d", telegramId)) {
				existingUser = &panelUser
			}
		}
		if existingUser == nil {
			existingUser = &v.GetResponse()[0]
		}
		return r.updateUser(ctx, existingUser, trafficLimit, days)
	default:
		return nil, errors.New("unknown response type")
	}
}

func (r *Client) updateUser(ctx context.Context, existingUser *remapi.UserDto, trafficLimit int, days int) (*remapi.UserDto, error) {

	newExpire := getNewExpire(days, existingUser.ExpireAt)

	userUpdate := &remapi.UpdateUserRequestDto{
		UUID:              existingUser.UUID,
		ExpireAt:          remapi.NewOptDateTime(newExpire),
		Status:            remapi.NewOptUpdateUserRequestDtoStatus(remapi.UpdateUserRequestDtoStatusACTIVE),
		TrafficLimitBytes: remapi.NewOptInt(trafficLimit),
	}

	var username string
	if ctx.Value("username") != nil {
		username = ctx.Value("username").(string)
		userUpdate.Description = remapi.NewOptNilString(username)
	} else {
		username = ""
	}

	updateUser, err := r.client.UsersControllerUpdateUser(ctx, userUpdate)
	if err != nil {
		return nil, err
	}
	tgid, _ := existingUser.TelegramId.Get()
	slog.Info("updated user", "telegramId", utils.MaskHalf(strconv.Itoa(tgid)), "username", utils.MaskHalf(username), "days", days)
	return &updateUser.Response, nil
}

func (r *Client) createUser(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int) (*remapi.UserDto, error) {
	expireAt := time.Now().UTC().AddDate(0, 0, days)
	username := generateUsername(customerId, telegramId)

	resp, err := r.client.InboundsControllerGetInbounds(ctx)
	if err != nil {
		return nil, err
	}

	inbounds := resp.GetResponse()
	inboundsId := make([]uuid.UUID, 0, len(config.InboundUUIDs()))
	for _, inbound := range inbounds {
		if config.InboundUUIDs() != nil && len(config.InboundUUIDs()) > 0 {
			if _, isExist := config.InboundUUIDs()[inbound.UUID]; !isExist {
				continue
			} else {
				inboundsId = append(inboundsId, inbound.UUID)
			}
		} else {
			inboundsId = append(inboundsId, inbound.UUID)
		}
	}

	createUserRequestDto := remapi.CreateUserRequestDto{
		Username:             username,
		ActiveUserInbounds:   inboundsId,
		Status:               remapi.NewOptCreateUserRequestDtoStatus(remapi.CreateUserRequestDtoStatusACTIVE),
		TelegramId:           remapi.NewOptInt(int(telegramId)),
		ExpireAt:             expireAt,
		TrafficLimitStrategy: remapi.CreateUserRequestDtoTrafficLimitStrategyMONTH,
		TrafficLimitBytes:    remapi.NewOptInt(trafficLimit),
	}

	var tgUsername string
	if ctx.Value("username") != nil {
		tgUsername = ctx.Value("username").(string)
		createUserRequestDto.Description = remapi.NewOptString(ctx.Value("username").(string))
	} else {
		tgUsername = ""
	}

	userCreate, err := r.client.UsersControllerCreateUser(ctx, &createUserRequestDto)
	if err != nil {
		return nil, err
	}
	slog.Info("created user", "telegramId", utils.MaskHalf(strconv.FormatInt(telegramId, 10)), "username", utils.MaskHalf(tgUsername), "days", days)
	return &userCreate.Response, nil
}

func (r *Client) GetUserByTelegramID(ctx context.Context, telegramId int64) (*remapi.UserDto, error) {
	resp, err := r.client.UsersControllerGetUserByTelegramId(ctx, remapi.UsersControllerGetUserByTelegramIdParams{TelegramId: strconv.FormatInt(telegramId, 10)})
	if err != nil {
		return nil, err
	}
	switch v := resp.(type) {
	case *remapi.UsersDto:
		if len(v.GetResponse()) == 0 {
			return nil, nil
		}
		for _, u := range v.GetResponse() {
			if strings.Contains(u.Username, fmt.Sprintf("_%d", telegramId)) {
				return &u, nil
			}
		}
		return &v.GetResponse()[0], nil
	default:
		return nil, nil
	}
}

func (r *Client) GetUserDailyUsage(ctx context.Context, uuid string, start, end time.Time) (float64, error) {
	resp, err := r.client.UsersStatsControllerGetUserUsageByRange(ctx, remapi.UsersStatsControllerGetUserUsageByRangeParams{UUID: uuid, Start: start, End: end})
	if err != nil {
		return 0, err
	}
	switch v := resp.(type) {
	case *remapi.GetUserUsageByRangeResponseDto:
		var total float64
		for _, item := range v.GetResponse() {
			total += item.Total
		}
		return total, nil
	default:
		return 0, nil
	}
}

func generateUsername(customerId int64, telegramId int64) string {
	return fmt.Sprintf("%d_%d", customerId, telegramId)
}

func getNewExpire(daysToAdd int, currentExpire time.Time) time.Time {
	if currentExpire.IsZero() {
		return time.Now().UTC().AddDate(0, 0, daysToAdd)
	}

	if currentExpire.Before(time.Now().UTC()) {
		return time.Now().UTC().AddDate(0, 0, daysToAdd)
	}

	return currentExpire.AddDate(0, 0, daysToAdd)
}
