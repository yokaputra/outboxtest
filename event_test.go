package outboxtest

import "testing"

func TestEventCloneCopiesMutableFields(t *testing.T) {
	original := Event{
		ID:             "evt_123",
		Type:           "order.created",
		IdempotencyKey: "order_123",
		Payload:        []byte(`{"order_id":"123"}`),
		Headers:        map[string]string{"trace_id": "trace_123"},
	}

	clone := original.Clone()
	original.Payload[0] = 'X'
	original.Headers["trace_id"] = "changed"

	if got, want := string(clone.Payload), `{"order_id":"123"}`; got != want {
		t.Fatalf("clone payload = %q, want %q", got, want)
	}
	if got, want := clone.Headers["trace_id"], "trace_123"; got != want {
		t.Fatalf("clone trace_id = %q, want %q", got, want)
	}
}
