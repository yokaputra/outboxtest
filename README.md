# outboxtest

`outboxtest` is a Go testkit for event-delivery reliability. It replaces an
application's publisher with a controllable fake so tests can model retries,
lost responses, duplicate delivery, slow destinations, and concurrent attempts
without a real message broker or arbitrary sleeps.

## What it tests

It helps test a delivery boundary such as an HTTP webhook sender, Kafka
producer, NATS publisher, or custom outbox worker.

It does **not** implement a production transactional outbox and does **not**
guarantee exactly-once delivery. An ambiguous result can cause a duplicate
delivery; the application consumer must still be idempotent.

## Install

```bash
go get github.com/yokaputra/outboxtest
```

## Quick start

Define an application boundary that a real transport and the test fake can both
implement:

```go
type Publisher interface {
    Publish(ctx context.Context, event outboxtest.Event) error
}
```

In a test, replace the real publisher with the kit's publisher:

```go
func TestOrderCreatedSurvivesAnAmbiguousPublish(t *testing.T) {
    kit := outboxtest.New(t,
        outboxtest.AcceptThenReturnErrorOnce(context.DeadlineExceeded),
    )

    worker := newWorker(store, kit.Publisher()) // worker belongs to the app
    createOrderAndEvent(t, store, order)

    worker.RunOnce(ctx) // destination accepts the event; caller sees an error
    worker.RunOnce(ctx) // application retries

    kit.AssertAttempts("order.created", 2)
    kit.AssertAccepted(order.EventID, 2)
    assertConsumerSideEffectCount(t, order.ID, 1)
}
```

The final assertion is intentionally owned by the application: the testkit can
show duplicate delivery, but only the application knows what its durable side
effect should be.

## Fault scenarios

| Option | Models | Acceptance recorded? |
|---|---|---|
| `ReturnErrorOnce(err)` | Destination fails before receiving the event | No |
| `AcceptThenReturnErrorOnce(err)` | Destination accepts event but response is lost | Yes |
| `DelayOnce(duration)` | Slow destination; honors context cancellation | On completion only |
| `BlockOnce(gate)` | Paused delivery for deterministic concurrency tests | On release only |

`New` consumes fault options in the supplied order. Once no scripted faults
remain, further publishes succeed and are accepted.

## Concurrency example

```go
gate := outboxtest.NewGate()
kit := outboxtest.New(t, outboxtest.BlockOnce(gate))

go worker.RunOnce(ctx)
<-gate.Started() // first delivery is now paused

worker.RunOnce(ctx) // exercise a concurrent attempt or claim behavior
gate.Release()
```

`Gate` prevents reliance on timing-sensitive `time.Sleep` calls. The core fake
publisher is safe to use with `go test -race`.

## Scope

- The initial library uses only the standard library.
- Broker- and database-specific adapters are deliberately postponed until a
  real integration need exists.

## Development

```bash
go test -race ./...
```
