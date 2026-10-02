package outboxtest

import (
	"context"
	"sync"
	"testing"
)

// Delivery is one call made to the fake publisher. Event is copied before it
// is recorded, so it remains trustworthy after Publish returns.
type Delivery struct {
	Sequence int
	Attempt  int
	Event    Event
	Accepted bool
}

// Kit controls and observes a Publisher used by an application's test.
type Kit struct {
	t         testing.TB
	publisher *fakePublisher
}

// Option configures a Kit.
type Option func(*fakePublisher)

// New creates a testkit. Fault options are consumed in the order supplied.
// A call to Publish with no remaining scripted fault is accepted successfully.
func New(t testing.TB, options ...Option) *Kit {
	t.Helper()

	publisher := &fakePublisher{
		attempts: make(map[string]int),
	}
	for _, option := range options {
		option(publisher)
	}

	return &Kit{t: t, publisher: publisher}
}

// Publisher returns the fake delivery boundary supplied to the application
// under test.
func (k *Kit) Publisher() Publisher {
	return k.publisher
}

// Deliveries returns a snapshot of every recorded delivery attempt.
func (k *Kit) Deliveries() []Delivery {
	return k.publisher.deliveriesSnapshot()
}

type fakePublisher struct {
	mu         sync.Mutex
	attempts   map[string]int
	deliveries []Delivery
	faults     []deliveryOutcome
}

func (p *fakePublisher) Publish(_ context.Context, event Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	attempt := p.attempts[event.ID] + 1
	p.attempts[event.ID] = attempt

	outcome := deliveryOutcome{accepted: true}
	if len(p.faults) > 0 {
		outcome = p.faults[0]
		p.faults = p.faults[1:]
	}

	p.deliveries = append(p.deliveries, Delivery{
		Sequence: len(p.deliveries) + 1,
		Attempt:  attempt,
		Event:    event.Clone(),
		Accepted: outcome.accepted,
	})

	return outcome.err
}

func (p *fakePublisher) deliveriesSnapshot() []Delivery {
	p.mu.Lock()
	defer p.mu.Unlock()

	snapshot := make([]Delivery, len(p.deliveries))
	for index, delivery := range p.deliveries {
		snapshot[index] = delivery
		snapshot[index].Event = delivery.Event.Clone()
	}
	return snapshot
}
