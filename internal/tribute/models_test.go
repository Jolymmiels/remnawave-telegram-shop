package tribute

import (
	"encoding/json"
	"testing"
)

func TestSubscriptionWebhookUnmarshalTrbUserID(t *testing.T) {
	var webhook SubscriptionWebhook
	data := []byte(`{"name":"new_subscription","payload":{"trb_user_id":"usr_123","telegram_user_id":123456789}}`)

	if err := json.Unmarshal(data, &webhook); err != nil {
		t.Fatalf("unmarshal webhook: %v", err)
	}
	if webhook.Payload.TrbUserID != "usr_123" {
		t.Fatalf("expected trb_user_id to be decoded, got %q", webhook.Payload.TrbUserID)
	}
	if webhook.Payload.TelegramUserID != 123456789 {
		t.Fatalf("expected telegram_user_id to be decoded, got %d", webhook.Payload.TelegramUserID)
	}
}
