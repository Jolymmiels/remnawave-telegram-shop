package wata

import "fmt"

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("wata API error: status=%d, body=%s", e.StatusCode, e.Body)
}
