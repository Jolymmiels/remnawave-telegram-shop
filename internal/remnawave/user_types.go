package remnawave

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// User represents a Remnawave user (API v3.4.1, UserResponseDto).
// Since v3 the panel identifies users by numeric ID (the legacy UUID
// field no longer exists); unused response fields are not decoded.
type User struct {
	ID                int64     `json:"id"`
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

// getUsersStreamResponse is the raw API response for GET /api/users/stream.
type getUsersStreamResponse struct {
	Users []User `json:"users"`
	// NextCursor is the cursor for the next page. The v3.4.1 schema declares
	// it as a JSON string or null, so it is decoded raw and parsed on demand.
	NextCursor json.RawMessage `json:"nextCursor"`
	HasMore    bool            `json:"hasMore"`
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

// CreateUserRequest is the request body for POST /api/users (CreateUserBodyDto).
// Per @remnawave/backend-contract@3.4.1: telegramId and description are
// scalar values (number / string). The published openapi.json shows them as
// arrays — that is a spec-generation artifact, do not "fix" this back.
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

// UpdateUserRequest is the request body for PATCH /api/users (UpdateUserBodyDto).
// v3.4.1 requires exactly one of ID or Username to identify the user;
// the legacy "uuid" field is no longer accepted. This client identifies
// users by ID only, so no Username field is exposed.
type UpdateUserRequest struct {
	ID                   *int64      `json:"id,omitempty"`
	Status               string      `json:"status,omitempty"`
	ExpireAt             *time.Time  `json:"expireAt,omitempty"`
	TrafficLimitBytes    *int        `json:"trafficLimitBytes,omitempty"`
	TrafficLimitStrategy string      `json:"trafficLimitStrategy,omitempty"`
	ActiveInternalSquads []uuid.UUID `json:"activeInternalSquads,omitempty"`
	ExternalSquadUuid    *uuid.UUID  `json:"externalSquadUuid,omitempty"`
	Tag                  *string     `json:"tag,omitempty"`
	Description          *string     `json:"description,omitempty"`
}
