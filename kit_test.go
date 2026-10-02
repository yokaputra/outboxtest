package outboxtest

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPublisherAcceptsEventsByDefault(t *testing.T) {
	kit := New(t)
	event := Event{ID: "evt_1", Type: "order.created"}

	if err := kit.Publisher().Publish(context.Background(), event); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	deliveries := kit.Deliveries()
	if len(deliveries) != 1 {
		t.Fatalf("recorded deliveries = %d, want 1", len(deliveries))
	}
	if !deliveries[0].Accepted {
		t.Fatal("delivery was not accepted")
	}
}

func TestPublisherRecordsFailureBeforeAcceptance(t *testing.T) {
	errTemporary := errors.New("temporary network error")
	kit := New(t, ReturnErrorOnce(errTemporary))
	event := Event{ID: "evt_1", Type: "order.created"}

	if err := kit.Publisher().Publish(context.Background(), event); !errors.Is(err, errTemporary) {
		t.Fatalf("Publish() error = %v, want %v", err, errTemporary)
	}

	deliveries := kit.Deliveries()
	if deliveries[0].Accepted {
		t.Fatal("delivery was accepted, want rejected")
	}
}

func TestPublisherRecordsAmbiguousAcceptance(t *testing.T) {
	errLostResponse := errors.New("response lost")
	kit := New(t, AcceptThenReturnErrorOnce(errLostResponse))
	event := Event{ID: "evt_1", Type: "order.created"}

	if err := kit.Publisher().Publish(context.Background(), event); !errors.Is(err, errLostResponse) {
		t.Fatalf("first Publish() error = %v, want %v", err, errLostResponse)
	}
	if err := kit.Publisher().Publish(context.Background(), event); err != nil {
		t.Fatalf("second Publish() error = %v", err)
	}

	deliveries := kit.Deliveries()
	if len(deliveries) != 2 {
		t.Fatalf("recorded deliveries = %d, want 2", len(deliveries))
	}
	if !deliveries[0].Accepted || !deliveries[1].Accepted {
		t.Fatal("both deliveries should be accepted")
	}
	if got, want := deliveries[1].Attempt, 2; got != want {
		t.Fatalf("second attempt = %d, want %d", got, want)
	}
}

func TestAssertionsAcceptMatchingDeliveryLog(t *testing.T) {
	errTemporary := errors.New("temporary network error")
	kit := New(t, ReturnErrorOnce(errTemporary))
	event := Event{ID: "evt_1", Type: "order.created"}

	_ = kit.Publisher().Publish(context.Background(), event)
	if err := kit.Publisher().Publish(context.Background(), event); err != nil {
		t.Fatalf("second Publish() error = %v", err)
	}

	kit.AssertAttempts("order.created", 2)
	kit.AssertAccepted("evt_1", 1)
	kit.AssertDeliveryOrder("order.created", "order.created")
}

func TestPublisherDelayHonorsContextCancellation(t *testing.T) {
	kit := New(t, DelayOnce(time.Hour))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := kit.Publisher().Publish(ctx, Event{ID: "evt_1", Type: "order.created"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Publish() error = %v, want %v", err, context.Canceled)
	}

	deliveries := kit.Deliveries()
	if deliveries[0].Accepted {
		t.Fatal("cancelled delivery was accepted")
	}
}

func TestPublisherGateControlsConcurrentAttempts(t *testing.T) {
	gate := NewGate()
	kit := New(t, BlockOnce(gate))
	firstDone := make(chan error, 1)

	go func() {
		firstDone <- kit.Publisher().Publish(context.Background(), Event{ID: "evt_1", Type: "order.created"})
	}()

	select {
	case <-gate.Started():
	case <-time.After(time.Second):
		t.Fatal("first publisher did not reach gate")
	}

	if err := kit.Publisher().Publish(context.Background(), Event{ID: "evt_2", Type: "order.updated"}); err != nil {
		t.Fatalf("second Publish() error = %v", err)
	}

	gate.Release()
	if err := <-firstDone; err != nil {
		t.Fatalf("first Publish() error = %v", err)
	}

	kit.AssertDeliveryOrder("order.created", "order.updated")
	kit.AssertAccepted("evt_1", 1)
	kit.AssertAccepted("evt_2", 1)
}
