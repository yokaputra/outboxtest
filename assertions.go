package outboxtest

import (
	"fmt"
	"strings"
)

// AssertAttempts reports a test error unless eventType was published exactly
// want times. It counts accepted and rejected attempts.
func (k *Kit) AssertAttempts(eventType string, want int) {
	k.t.Helper()

	got := 0
	for _, delivery := range k.Deliveries() {
		if delivery.Event.Type == eventType {
			got++
		}
	}
	if got != want {
		k.t.Errorf("delivery attempts for event type %q = %d, want %d\nrecorded deliveries:\n%s", eventType, got, want, formatDeliveries(k.Deliveries()))
	}
}

// AssertAccepted reports a test error unless eventID was accepted exactly want
// times. An accepted event that later caused a client-side error still counts.
func (k *Kit) AssertAccepted(eventID string, want int) {
	k.t.Helper()

	got := 0
	for _, delivery := range k.Deliveries() {
		if delivery.Event.ID == eventID && delivery.Accepted {
			got++
		}
	}
	if got != want {
		k.t.Errorf("accepted deliveries for event ID %q = %d, want %d\nrecorded deliveries:\n%s", eventID, got, want, formatDeliveries(k.Deliveries()))
	}
}

// AssertDeliveryOrder reports a test error unless the delivery log has exactly
// the supplied event types in order.
func (k *Kit) AssertDeliveryOrder(eventTypes ...string) {
	k.t.Helper()

	deliveries := k.Deliveries()
	if len(deliveries) != len(eventTypes) {
		k.t.Errorf("recorded deliveries = %d, want %d\nrecorded deliveries:\n%s", len(deliveries), len(eventTypes), formatDeliveries(deliveries))
		return
	}

	for index, delivery := range deliveries {
		if got, want := delivery.Event.Type, eventTypes[index]; got != want {
			k.t.Errorf("delivery at position %d has type %q, want %q\nrecorded deliveries:\n%s", index+1, got, want, formatDeliveries(deliveries))
			return
		}
	}
}

func formatDeliveries(deliveries []Delivery) string {
	if len(deliveries) == 0 {
		return "  (none)"
	}

	lines := make([]string, 0, len(deliveries))
	for _, delivery := range deliveries {
		lines = append(lines, fmt.Sprintf(
			"  #%d attempt=%d accepted=%t event_id=%q event_type=%q",
			delivery.Sequence,
			delivery.Attempt,
			delivery.Accepted,
			delivery.Event.ID,
			delivery.Event.Type,
		))
	}

	return strings.Join(lines, "\n")
}
