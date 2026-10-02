// Package outboxtest provides test doubles and assertions for event-delivery
// reliability tests.
package outboxtest

import "context"

// Event is the transport-neutral envelope supplied to a Publisher.
//
// ID identifies one event occurrence. IdempotencyKey identifies the stable
// operation a destination or consumer may use to deduplicate delivery.
type Event struct {
	ID             string
	Type           string
	IdempotencyKey string
	Payload        []byte
	Headers        map[string]string
}

// Clone returns an independent copy of e. It is used when a test double keeps
// a delivery record so later caller mutation cannot change test evidence.
func (e Event) Clone() Event {
	clone := e
	clone.Payload = append([]byte(nil), e.Payload...)

	if e.Headers != nil {
		clone.Headers = make(map[string]string, len(e.Headers))
		for key, value := range e.Headers {
			clone.Headers[key] = value
		}
	}

	return clone
}

// Publisher is the delivery boundary an application replaces with a testkit
// publisher in tests. Implementations may use HTTP, a message broker, or any
// other destination.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
