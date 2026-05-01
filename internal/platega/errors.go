package platega

import "fmt"

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("platega API error: status=%d, body=%s", e.StatusCode, e.Body)
}

func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

func (e *APIError) IsBadRequest() bool {
	return e.StatusCode == 400
}
