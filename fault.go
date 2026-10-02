package outboxtest

import (
	"context"
	"sync"
	"time"
)

type deliveryOutcome struct {
	accepted bool
	err      error
	delay    time.Duration
	gate     *Gate
}

// ReturnErrorOnce makes the next Publish call fail before the fake destination
// accepts its event. It models a temporary delivery failure.
func ReturnErrorOnce(err error) Option {
	return appendOutcome(deliveryOutcome{err: err})
}

// AcceptThenReturnErrorOnce makes the next Publish call record acceptance but
// return err to the caller. It models a lost response or another ambiguous
// result: retrying the event can create a duplicate delivery.
func AcceptThenReturnErrorOnce(err error) Option {
	return appendOutcome(deliveryOutcome{accepted: true, err: err})
}

// DelayOnce delays the next Publish call before the fake destination accepts
// it. The delay honors context cancellation.
func DelayOnce(delay time.Duration) Option {
	return appendOutcome(deliveryOutcome{accepted: true, delay: delay})
}

// Gate blocks a fake delivery until Release is called. A Gate is useful for
// deterministic concurrent-worker tests without arbitrary sleeps.
type Gate struct {
	started     chan struct{}
	released    chan struct{}
	startOnce   sync.Once
	releaseOnce sync.Once
}

// NewGate creates a Gate in its blocked state.
func NewGate() *Gate {
	return &Gate{
		started:  make(chan struct{}),
		released: make(chan struct{}),
	}
}

// Started is closed when a publisher begins waiting on g.
func (g *Gate) Started() <-chan struct{} {
	return g.started
}

// Release unblocks every publisher waiting on g. It is safe to call more than
// once.
func (g *Gate) Release() {
	g.releaseOnce.Do(func() { close(g.released) })
}

func (g *Gate) wait(ctx context.Context) error {
	g.startOnce.Do(func() { close(g.started) })

	select {
	case <-g.released:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// BlockOnce blocks the next Publish call on gate before the fake destination
// accepts it. The caller releases it with gate.Release().
func BlockOnce(gate *Gate) Option {
	if gate == nil {
		panic("outboxtest: BlockOnce requires a non-nil gate")
	}

	return appendOutcome(deliveryOutcome{accepted: true, gate: gate})
}

func appendOutcome(outcome deliveryOutcome) Option {
	return func(publisher *fakePublisher) {
		publisher.faults = append(publisher.faults, outcome)
	}
}

func (outcome deliveryOutcome) wait(ctx context.Context) error {
	if outcome.gate != nil {
		if err := outcome.gate.wait(ctx); err != nil {
			return err
		}
	}
	if outcome.delay <= 0 {
		return nil
	}

	timer := time.NewTimer(outcome.delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
