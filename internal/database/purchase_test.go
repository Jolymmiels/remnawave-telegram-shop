package database

import (
	"context"
	"testing"
)

func TestFindLatestActiveTributesByCustomerIDsEmpty(t *testing.T) {
	repo := &PurchaseRepository{}

	result, err := repo.FindLatestActiveTributesByCustomerIDs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatalf("result should not be nil")
	}

	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d", len(result))
	}
}
