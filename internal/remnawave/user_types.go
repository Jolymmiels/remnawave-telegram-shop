package remnawave

import (
	"time"

	"github.com/google/uuid"
)

// User represents a Remnawave user.
type User struct {
	UUID              uuid.UUID `json:"uuid"`
	Username          string    `json:"username"`
	SubscriptionUrl   string    `json:"subscriptionUrl"`
	ExpireAt          time.Time `json:"expireAt"`
	TelegramID        *int64    `json:"telegramId"`
	Status            string    `json:"status"`
	TrafficLimitBytes int       `json:"trafficLimitBytes"`
}

// getAllUsersResponse is the raw API response for GET /api/users.
type getAllUsersResponse struct {
	Response struct {
		Users []User `json:"users"`
		Total int    `json:"total"`
	} `json:"response"`
}

// apiResponse is a generic wrapper for { "response": T } API responses.
type apiResponse[T any] struct {
	Response T `json:"response"`
}

// apiErrorResponse is the standard error response from the Remnawave API.
type apiErrorResponse struct {
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
}

// internalSquadItem is a single squad in the internal squads response.
type internalSquadItem struct {
	UUID uuid.UUID `json:"uuid"`
	Name string    `json:"name"`
}

// internalSquadsResponse is the response body for GET /api/internal-squads.
type internalSquadsResponse struct {
	InternalSquads []internalSquadItem `json:"internalSquads"`
}

// CreateUserRequest is the request body for POST /api/users.
type CreateUserRequest struct {
	Username             string      `json:"username"`
	ExpireAt             time.Time   `json:"expireAt"`
	Status               string      `json:"status,omitempty"`
	TrafficLimitBytes    *int        `json:"trafficLimitBytes,omitempty"`
	TrafficLimitStrategy string      `json:"trafficLimitStrategy,omitempty"`
	ActiveInternalSquads []uuid.UUID `json:"activeInternalSquads,omitempty"`
	ExternalSquadUuid    *uuid.UUID  `json:"externalSquadUuid,omitempty"`
	Tag                  *string     `json:"tag,omitempty"`
	TelegramID           *int64      `json:"telegramId,omitempty"`
	Description          *string     `json:"description,omitempty"`
}

// UpdateUserRequest is the request body for PATCH /api/users.
type UpdateUserRequest struct {
	UUID                 *uuid.UUID  `json:"uuid,omitempty"`
	Status               string      `json:"status,omitempty"`
	ExpireAt             *time.Time  `json:"expireAt,omitempty"`
	TrafficLimitBytes    *int        `json:"trafficLimitBytes,omitempty"`
	TrafficLimitStrategy string      `json:"trafficLimitStrategy,omitempty"`
	ActiveInternalSquads []uuid.UUID `json:"activeInternalSquads,omitempty"`
	ExternalSquadUuid    *uuid.UUID  `json:"externalSquadUuid,omitempty"`
	Tag                  *string     `json:"tag,omitempty"`
	Description          *string     `json:"description,omitempty"`
}
