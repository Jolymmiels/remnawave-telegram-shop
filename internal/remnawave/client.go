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
	params := remapi.UsersControllerGetAllUsersParams{
		Size:  remapi.NewOptFloat64(1),
		Start: remapi.NewOptFloat64(0),
	}
	_, err := r.client.UsersControllerGetAllUsers(ctx, params)
	return err
}

func (r *Client) GetUsers(ctx context.Context) (*[]remapi.GetAllUsersResponseDtoResponseUsersItem, error) {
	pager := remapi.NewPaginationHelper(250)
	users := make([]remapi.GetAllUsersResponseDtoResponseUsersItem, 0)

	for {
		params := remapi.UsersControllerGetAllUsersParams{
			Start: remapi.NewOptFloat64(float64(pager.Offset)),
			Size:  remapi.NewOptFloat64(float64(pager.Limit)),
		}

		resp, err := r.client.Users().GetAllUsers(ctx, params)
		if err != nil {
			return nil, err
		}

		response := resp.(*remapi.GetAllUsersResponseDto).GetResponse()
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

func (r *Client) DecreaseSubscription(ctx context.Context, telegramId int64, trafficLimit, days int) (*time.Time, error) {
	resp, err := r.client.Users().GetUserByTelegramId(ctx, remapi.UsersControllerGetUserByTelegramIdParams{TelegramId: strconv.FormatInt(telegramId, 10)})
	if err != nil {
		return nil, err
	}

	switch v := resp.(type) {
	case *remapi.UsersControllerGetUserByTelegramIdNotFound:
		return nil, errors.New("user in remnawave not found")
	case *remapi.UsersResponse:
		var existingUser *remapi.UsersResponseResponseItem
		for _, panelUser := range v.GetResponse() {
			if strings.Contains(panelUser.Username, fmt.Sprintf("_%d", telegramId)) {
				existingUser = &panelUser
			}
		}
		if existingUser == nil {
			existingUser = &v.GetResponse()[0]
		}
		updatedUser, err := r.updateUser(ctx, existingUser, trafficLimit, days)
		return &updatedUser.ExpireAt, err
	default:
		return nil, errors.New("unknown response type")
	}
}

func (r *Client) CreateOrUpdateUser(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int, isTrialUser bool) (*remapi.UserResponseResponse, error) {
	return r.CreateOrUpdateUserWithPlan(ctx, customerId, telegramId, trafficLimit, days, isTrialUser, nil)
}

func (r *Client) CreateOrUpdateUserWithPlan(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int, isTrialUser bool, plan *database.Plan) (*remapi.UserResponseResponse, error) {
	resp, err := r.client.UsersControllerGetUserByTelegramId(ctx, remapi.UsersControllerGetUserByTelegramIdParams{TelegramId: strconv.FormatInt(telegramId, 10)})
	if err != nil {
		return nil, err
	}

	switch v := resp.(type) {

	case *remapi.UsersControllerGetUserByTelegramIdNotFound:
		return r.createUserWithPlan(ctx, customerId, telegramId, trafficLimit, days, isTrialUser, plan)
	case *remapi.UsersResponse:
		var existingUser *remapi.UsersResponseResponseItem
		for _, panelUser := range v.GetResponse() {
			if strings.Contains(panelUser.Username, fmt.Sprintf("_%d", telegramId)) {
				existingUser = &panelUser
			}
		}
		if existingUser == nil {
			existingUser = &v.GetResponse()[0]
		}
		return r.updateUserWithPlan(ctx, existingUser, trafficLimit, days, isTrialUser, plan)
	default:
		return nil, errors.New("unknown response type")
	}
}

func (r *Client) updateUser(ctx context.Context, existingUser *remapi.UsersResponseResponseItem, trafficLimit int, days int) (*remapi.UserResponseResponse, error) {
	return r.updateUserWithPlan(ctx, existingUser, trafficLimit, days, false, nil)
}

func (r *Client) updateUserWithPlan(ctx context.Context, existingUser *remapi.UsersResponseResponseItem, trafficLimit int, days int, isTrialUser bool, plan *database.Plan) (*remapi.UserResponseResponse, error) {
	newExpire := getNewExpire(days, existingUser.ExpireAt)

	resp, err := r.client.InternalSquadControllerGetInternalSquads(ctx)
	if err != nil {
		return nil, err
	}

	squads := resp.(*remapi.GetInternalSquadsResponseDto).GetResponse()

	// Determine selected squads based on plan, trial, or config
	selectedSquads := config.SquadUUIDs()
	if isTrialUser {
		selectedSquads = config.TrialInternalSquads()
	} else if plan != nil && plan.InternalSquads != "" {
		selectedSquads = parseSquadUUIDs(plan.InternalSquads)
	}

	squadId := make([]uuid.UUID, 0, len(selectedSquads))
	for _, squad := range squads.GetInternalSquads() {
		if selectedSquads != nil && len(selectedSquads) > 0 {
			if _, isExist := selectedSquads[squad.UUID]; !isExist {
				continue
			} else {
				squadId = append(squadId, squad.UUID)
			}
		} else {
			squadId = append(squadId, squad.UUID)
		}
	}

	// Determine traffic limit strategy
	strategy := config.TrafficLimitResetStrategy()
	if isTrialUser {
		strategy = config.TrialTrafficLimitResetStrategy()
	}

	userUpdate := &remapi.UpdateUserRequestDto{
		UUID:                 remapi.NewOptUUID(existingUser.UUID),
		ExpireAt:             remapi.NewOptDateTime(newExpire),
		Status:               remapi.NewOptUpdateUserRequestDtoStatus(remapi.UpdateUserRequestDtoStatusACTIVE),
		TrafficLimitBytes:    remapi.NewOptInt(trafficLimit),
		ActiveInternalSquads: squadId,
		TrafficLimitStrategy: remapi.NewOptUpdateUserRequestDtoTrafficLimitStrategy(getUpdateStrategy(strategy)),
	}

	// Determine external squad based on plan, trial, or config
	externalSquad := config.ExternalSquadUUID()
	if isTrialUser {
		externalSquad = config.TrialExternalSquadUUID()
	} else if plan != nil && plan.ExternalSquadUUID != "" {
		if parsed, err := uuid.Parse(plan.ExternalSquadUUID); err == nil {
			externalSquad = parsed
		}
	}
	if externalSquad != uuid.Nil {
		userUpdate.ExternalSquadUuid = remapi.NewOptNilUUID(externalSquad)
	}

	// Determine tag based on plan, trial, or config
	tag := config.RemnawaveTag()
	if isTrialUser {
		tag = config.TrialRemnawaveTag()
	} else if plan != nil && plan.RemnawaveTag != "" {
		tag = plan.RemnawaveTag
	}
	if tag != "" {
		userUpdate.Tag = remapi.NewOptNilString(tag)
	}

	if plan != nil && plan.DeviceLimit != nil {
		userUpdate.HwidDeviceLimit = remapi.NewOptNilInt(*plan.DeviceLimit)
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
	if value, ok := updateUser.(*remapi.UsersControllerUpdateUserInternalServerError); ok {
		return nil, errors.New("error while updating user. message: " + value.GetMessage().Value + ". code: " + value.GetErrorCode().Value)
	}

	tgid, _ := existingUser.TelegramId.Get()
	slog.Info("updated user", "telegramId", utils.MaskHalf(strconv.Itoa(tgid)), "username", utils.MaskHalf(username), "days", days)
	return &updateUser.(*remapi.UserResponse).Response, nil
}

func (r *Client) createUser(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int, isTrialUser bool) (*remapi.UserResponseResponse, error) {
	return r.createUserWithPlan(ctx, customerId, telegramId, trafficLimit, days, isTrialUser, nil)
}

func (r *Client) createUserWithPlan(ctx context.Context, customerId int64, telegramId int64, trafficLimit int, days int, isTrialUser bool, plan *database.Plan) (*remapi.UserResponseResponse, error) {
	expireAt := time.Now().UTC().AddDate(0, 0, days)
	username := generateUsername(customerId, telegramId)

	resp, err := r.client.InternalSquadControllerGetInternalSquads(ctx)
	if err != nil {
		return nil, err
	}

	squads := resp.(*remapi.GetInternalSquadsResponseDto).GetResponse()

	// Determine selected squads based on plan, trial, or config
	selectedSquads := config.SquadUUIDs()
	if isTrialUser {
		selectedSquads = config.TrialInternalSquads()
	} else if plan != nil && plan.InternalSquads != "" {
		selectedSquads = parseSquadUUIDs(plan.InternalSquads)
	}

	squadId := make([]uuid.UUID, 0, len(selectedSquads))
	for _, squad := range squads.GetInternalSquads() {
		if selectedSquads != nil && len(selectedSquads) > 0 {
			if _, isExist := selectedSquads[squad.UUID]; !isExist {
				continue
			} else {
				squadId = append(squadId, squad.UUID)
			}
		} else {
			squadId = append(squadId, squad.UUID)
		}
	}

	// Determine external squad
	externalSquad := config.ExternalSquadUUID()
	if isTrialUser {
		externalSquad = config.TrialExternalSquadUUID()
	} else if plan != nil && plan.ExternalSquadUUID != "" {
		if parsed, err := uuid.Parse(plan.ExternalSquadUUID); err == nil {
			externalSquad = parsed
		}
	}

	strategy := config.TrafficLimitResetStrategy()
	if isTrialUser {
		strategy = config.TrialTrafficLimitResetStrategy()
	}

	createUserRequestDto := remapi.CreateUserRequestDto{
		Username:             username,
		ActiveInternalSquads: squadId,
		Status:               remapi.NewOptCreateUserRequestDtoStatus(remapi.CreateUserRequestDtoStatusACTIVE),
		TelegramId:           remapi.NewOptNilInt(int(telegramId)),
		ExpireAt:             expireAt,
		TrafficLimitStrategy: remapi.NewOptCreateUserRequestDtoTrafficLimitStrategy(getCreateStrategy(strategy)),
		TrafficLimitBytes:    remapi.NewOptInt(trafficLimit),
	}
	if externalSquad != uuid.Nil {
		createUserRequestDto.ExternalSquadUuid = remapi.NewOptNilUUID(externalSquad)
	}

	// Determine tag
	tag := config.RemnawaveTag()
	if isTrialUser {
		tag = config.TrialRemnawaveTag()
	} else if plan != nil && plan.RemnawaveTag != "" {
		tag = plan.RemnawaveTag
	}
	if tag != "" {
		createUserRequestDto.Tag = remapi.NewOptNilString(tag)
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
	return &userCreate.(*remapi.UserResponse).Response, nil
}

// parseSquadUUIDs parses comma-separated UUID string into a map
func parseSquadUUIDs(s string) map[uuid.UUID]uuid.UUID {
	result := make(map[uuid.UUID]uuid.UUID)
	if s == "" {
		return result
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if parsed, err := uuid.Parse(part); err == nil {
			result[parsed] = parsed
		}
	}
	return result
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

func getCreateStrategy(s string) remapi.CreateUserRequestDtoTrafficLimitStrategy {
	switch s {
	case "DAY":
		return remapi.CreateUserRequestDtoTrafficLimitStrategyDAY
	case "WEEK":
		return remapi.CreateUserRequestDtoTrafficLimitStrategyWEEK
	case "NO_RESET":
		return remapi.CreateUserRequestDtoTrafficLimitStrategyNORESET
	default:
		return remapi.CreateUserRequestDtoTrafficLimitStrategyMONTH
	}
}

func getUpdateStrategy(s string) remapi.UpdateUserRequestDtoTrafficLimitStrategy {
	switch s {
	case "DAY":
		return remapi.UpdateUserRequestDtoTrafficLimitStrategyDAY
	case "WEEK":
		return remapi.UpdateUserRequestDtoTrafficLimitStrategyWEEK
	case "NO_RESET":
		return remapi.UpdateUserRequestDtoTrafficLimitStrategyNORESET
	default:
		return remapi.UpdateUserRequestDtoTrafficLimitStrategyMONTH
	}
}

type Device struct {
	Hwid        string    `json:"hwid"`
	UserUuid    string    `json:"user_uuid"`
	Platform    string    `json:"platform"`
	OsVersion   string    `json:"os_version"`
	DeviceModel string    `json:"device_model"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (r *Client) GetUserUuidByTelegramId(ctx context.Context, telegramId int64) (string, error) {
	resp, err := r.client.Users().GetUserByTelegramId(ctx, remapi.UsersControllerGetUserByTelegramIdParams{
		TelegramId: strconv.FormatInt(telegramId, 10),
	})
	if err != nil {
		return "", err
	}

	switch v := resp.(type) {
	case *remapi.UsersControllerGetUserByTelegramIdNotFound:
		return "", errors.New("user not found")
	case *remapi.UsersResponse:
		users := v.GetResponse()
		if len(users) == 0 {
			return "", errors.New("user not found")
		}
		var targetUser *remapi.UsersResponseResponseItem
		for _, user := range users {
			if strings.Contains(user.Username, fmt.Sprintf("_%d", telegramId)) {
				targetUser = &user
				break
			}
		}
		if targetUser == nil {
			targetUser = &users[0]
		}
		return targetUser.UUID.String(), nil
	default:
		return "", errors.New("unknown response type")
	}
}

func (r *Client) GetUserDevices(ctx context.Context, userUuid string) ([]Device, error) {
	resp, err := r.client.HwidUserDevices().GetUserHwidDevices(ctx, remapi.HwidUserDevicesControllerGetUserHwidDevicesParams{
		UserUuid: userUuid,
	})
	if err != nil {
		return nil, err
	}

	switch v := resp.(type) {
	case *remapi.HwidDevicesResponse:
		devices := make([]Device, 0, len(v.Response.Devices))
		for _, d := range v.Response.Devices {
			platform, _ := d.Platform.Get()
			osVersion, _ := d.OsVersion.Get()
			deviceModel, _ := d.DeviceModel.Get()

			devices = append(devices, Device{
				Hwid:        d.Hwid,
				UserUuid:    d.UserUuid.String(),
				Platform:    platform,
				OsVersion:   osVersion,
				DeviceModel: deviceModel,
				CreatedAt:   d.CreatedAt,
				UpdatedAt:   d.UpdatedAt,
			})
		}
		return devices, nil
	default:
		return nil, errors.New("failed to get user devices")
	}
}

func (r *Client) DeleteUserDevice(ctx context.Context, userUuid, hwid string) error {
	userUuidParsed, err := uuid.Parse(userUuid)
	if err != nil {
		return errors.New("invalid user uuid")
	}

	resp, err := r.client.HwidUserDevices().DeleteUserHwidDevice(ctx, &remapi.DeleteUserHwidDeviceRequestDto{
		UserUuid: userUuidParsed,
		Hwid:     hwid,
	})
	if err != nil {
		return err
	}

	switch resp.(type) {
	case *remapi.HwidDevicesResponse:
		return nil
	case *remapi.HwidUserDevicesControllerDeleteUserHwidDeviceBadRequest:
		return errors.New("bad request")
	case *remapi.HwidUserDevicesControllerDeleteUserHwidDeviceInternalServerError:
		return errors.New("internal server error")
	default:
		return errors.New("failed to delete device")
	}
}

func (r *Client) DeleteAllUserDevices(ctx context.Context, userUuid string) error {
	userUuidParsed, err := uuid.Parse(userUuid)
	if err != nil {
		return errors.New("invalid user uuid")
	}

	resp, err := r.client.HwidUserDevices().DeleteAllUserHwidDevices(ctx, &remapi.DeleteAllUserHwidDevicesRequestDto{
		UserUuid: userUuidParsed,
	})
	if err != nil {
		return err
	}

	switch resp.(type) {
	case *remapi.HwidDevicesResponse:
		return nil
	case *remapi.HwidUserDevicesControllerDeleteAllUserHwidDevicesBadRequest:
		return errors.New("bad request")
	case *remapi.HwidUserDevicesControllerDeleteAllUserHwidDevicesInternalServerError:
		return errors.New("internal server error")
	default:
		return errors.New("failed to delete all devices")
	}
}

func (r *Client) RevokeUserSubscription(ctx context.Context, userUuid string) error {
	resp, err := r.client.Users().RevokeUserSubscription(ctx, &remapi.RevokeUserSubscriptionBodyDto{}, remapi.UsersControllerRevokeUserSubscriptionParams{
		UUID: userUuid,
	})
	if err != nil {
		return err
	}

	switch resp.(type) {
	case *remapi.UserResponse:
		return nil
	case *remapi.UsersControllerRevokeUserSubscriptionNotFound:
		return errors.New("user not found")
	case *remapi.UsersControllerRevokeUserSubscriptionBadRequest:
		return errors.New("bad request")
	case *remapi.UsersControllerRevokeUserSubscriptionInternalServerError:
		return errors.New("internal server error")
	default:
		return errors.New("failed to revoke subscription")
	}
}

// Squad represents an internal or external squad from Remnawave
type Squad struct {
	UUID uuid.UUID `json:"uuid"`
	Name string    `json:"name"`
}

// GetSquads returns all available internal squads from Remnawave
func (r *Client) GetSquads(ctx context.Context) ([]Squad, error) {
	resp, err := r.client.InternalSquadControllerGetInternalSquads(ctx)
	if err != nil {
		return nil, err
	}

	squadsResp := resp.(*remapi.GetInternalSquadsResponseDto).GetResponse()

	squads := make([]Squad, 0, len(squadsResp.GetInternalSquads()))
	for _, s := range squadsResp.GetInternalSquads() {
		squads = append(squads, Squad{
			UUID: s.UUID,
			Name: s.Name,
		})
	}

	return squads, nil
}

// GetExternalSquads returns all available external squads from Remnawave
func (r *Client) GetExternalSquads(ctx context.Context) ([]Squad, error) {
	resp, err := r.client.ExternalSquadControllerGetExternalSquads(ctx)
	if err != nil {
		return nil, err
	}

	squadsResp := resp.(*remapi.GetExternalSquadsResponseDto).GetResponse()

	squads := make([]Squad, 0, len(squadsResp.GetExternalSquads()))
	for _, s := range squadsResp.GetExternalSquads() {
		squads = append(squads, Squad{
			UUID: s.UUID,
			Name: s.Name,
		})
	}

	return squads, nil
}
