package remnawave

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoRequestIncludesValidationDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"statusCode":400,"message":"Validation failed","errors":[{"code":"invalid_format","path":["tag"],"message":"Tag can only contain uppercase letters, numbers, underscores"}]}`))
	}))
	defer server.Close()

	client := &Client{httpClient: server.Client(), baseURL: server.URL}
	_, status, err := client.doRequest(context.Background(), http.MethodPost, "/api/users", map[string]string{"tag": "invalid-tag"})

	if status != http.StatusBadRequest {
		t.Fatalf("unexpected status: want %d, got %d", http.StatusBadRequest, status)
	}
	if err == nil {
		t.Fatal("expected an API error")
	}
	for _, expected := range []string{"API error 400: Validation failed", `"path":["tag"]`, "Tag can only contain uppercase letters"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("error %q does not contain %q", err, expected)
		}
	}
	if strings.Contains(err.Error(), "code: )") {
		t.Fatalf("error contains an empty error code: %q", err)
	}
}

func TestDoRequestIncludesNonEmptyErrorCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"message":"Username already exists","errorCode":"E029"}`))
	}))
	defer server.Close()

	client := &Client{httpClient: server.Client(), baseURL: server.URL}
	_, _, err := client.doRequest(context.Background(), http.MethodPost, "/api/users", nil)

	if err == nil {
		t.Fatal("expected an API error")
	}
	if !strings.Contains(err.Error(), "(code: E029)") {
		t.Fatalf("error does not contain the API error code: %q", err)
	}
}
