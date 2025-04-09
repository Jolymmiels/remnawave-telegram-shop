package remnawave

import (
	"github.com/google/uuid"
	"time"
)

type Node struct {
	UUID                    string            `json:"uuid"`
	Name                    string            `json:"name"`
	Address                 string            `json:"address"`
	Port                    int               `json:"port"`
	IsConnected             bool              `json:"isConnected"`
	IsConnecting            bool              `json:"isConnecting"`
	IsDisabled              bool              `json:"isDisabled"`
	IsNodeOnline            bool              `json:"isNodeOnline"`
	IsXrayRunning           bool              `json:"isXrayRunning"`
	LastStatusChange        time.Time         `json:"lastStatusChange"`
	LastStatusMessage       string            `json:"lastStatusMessage"`
	XrayVersion             string            `json:"xrayVersion"`
	IsTrafficTrackingActive bool              `json:"isTrafficTrackingActive"`
	TrafficResetDay         int               `json:"trafficResetDay"`
	UsersOnline             int               `json:"usersOnline"`
	CpuCount                *int              `json:"cpuCount"`
	CpuModel                *string           `json:"cpuModel"`
	TotalRam                *string           `json:"totalRam"`
	ConsumptionMultiplier   float64           `json:"consumptionMultiplier"`
	TrafficLimitBytes       *int64            `json:"trafficLimitBytes"`
	TrafficUsedBytes        *int64            `json:"trafficUsedBytes"`
	NotifyPercent           *int              `json:"notifyPercent"`
	ViewPosition            int               `json:"viewPosition"`
	CountryCode             string            `json:"countryCode"`
	CreatedAt               time.Time         `json:"createdAt"`
	UpdatedAt               time.Time         `json:"updatedAt"`
	ExcludedInbounds        []ExcludedInbound `json:"excludedInbounds"`
}

type ExcludedInbound struct {
	UUID string `json:"uuid"`
	Tag  string `json:"tag"`
	Type string `json:"type"`
}
type UserUpdate struct {
	UUID                 string               `json:"uuid"`
	Status               Status               `json:"status,omitempty"`
	TrafficLimitBytes    int64                `json:"trafficLimitBytes"`
	TrafficLimitStrategy TrafficLimitStrategy `json:"trafficLimitStrategy,omitempty"`
	ActiveUserInbounds   []string             `json:"activeUserInbounds,omitempty"`
	ExpireAt             time.Time            `json:"expireAt,omitempty"`
	Description          string               `json:"description,omitempty"`
	TelegramId           int64                `json:"telegramId,omitempty"`
}
type User struct {
	UUID                     string               `json:"uuid"`
	SubscriptionUUID         string               `json:"subscriptionUuid"`
	ShortUUID                string               `json:"shortUuid"`
	Username                 string               `json:"username"`
	Status                   Status               `json:"status"`
	UsedTrafficBytes         int64                `json:"usedTrafficBytes"`
	LifetimeUsedTrafficBytes int64                `json:"lifetimeUsedTrafficBytes"`
	TrafficLimitBytes        int64                `json:"trafficLimitBytes"`
	TrafficLimitStrategy     TrafficLimitStrategy `json:"trafficLimitStrategy"`
	SubLastUserAgent         string               `json:"subLastUserAgent"`
	SubLastOpenedAt          time.Time            `json:"subLastOpenedAt"`
	ExpireAt                 time.Time            `json:"expireAt"`
	OnlineAt                 time.Time            `json:"onlineAt"`
	SubRevokedAt             time.Time            `json:"subRevokedAt"`
	LastTrafficResetAt       time.Time            `json:"lastTrafficResetAt"`
	TrojanPassword           string               `json:"trojanPassword"`
	VlessUUID                string               `json:"vlessUuid"`
	SSPassword               string               `json:"ssPassword"`
	Description              string               `json:"description"`
	CreatedAt                time.Time            `json:"createdAt"`
	UpdatedAt                time.Time            `json:"updatedAt"`
	ActiveUserInbounds       []Inbound            `json:"activeUserInbounds"`
	SubscriptionURL          string               `json:"subscriptionUrl"`
	TelegramId               *int64               `json:"telegramId"`
}

type UsersResponse struct {
	Total int    `json:"total"`
	Users []User `json:"users"`
}

type Inbound struct {
	UUID uuid.UUID `json:"uuid"`
	Tag  string    `json:"tag"`
	Type string    `json:"type"`
}

type ResponseWrapper[T any] struct {
	Response T `json:"response"`
}

type UserCreate struct {
	Username             string               `json:"username,omitempty"`
	Status               Status               `json:"status,omitempty"`
	SubscriptionUuid     *uuid.UUID           `json:"subscriptionUuid,omitempty"`
	ShortUuid            string               `json:"shortUuid,omitempty"`
	TrojanPassword       string               `json:"trojanPassword,omitempty"`
	VlessUuid            string               `json:"vlessUuid,omitempty"`
	SsPassword           string               `json:"ssPassword,omitempty"`
	TrafficLimitBytes    int64                `json:"trafficLimitBytes,omitempty"`
	TrafficLimitStrategy TrafficLimitStrategy `json:"trafficLimitStrategy,omitempty"`
	ActiveUserInbounds   []uuid.UUID          `json:"activeUserInbounds,omitempty"`
	ExpireAt             time.Time            `json:"expireAt,omitempty"`
	CreatedAt            *time.Time           `json:"createdAt,omitempty"`
	LastTrafficResetAt   *time.Time           `json:"lastTrafficResetAt,omitempty"`
	Description          string               `json:"description,omitempty"`
	ActivateAllInbounds  bool                 `json:"activateAllInbounds,omitempty"`
	TelegramId           int64                `json:"telegramId,omitempty"`
}

type Status string

const (
	ACTIVE   Status = "ACTIVE"
	DISABLED Status = "DISABLED"
	LIMITED  Status = "LIMITED"
	EXPIRED  Status = "EXPIRED"
)

type TrafficLimitStrategy string

const (
	NO_RESET TrafficLimitStrategy = "NO_RESET"
	DAY      TrafficLimitStrategy = "DAY"
	WEEK     TrafficLimitStrategy = "WEEK"
	MONTH    TrafficLimitStrategy = "MONTH"
)
