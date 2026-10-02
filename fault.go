package outboxtest

type deliveryOutcome struct {
	accepted bool
	err      error
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

func appendOutcome(outcome deliveryOutcome) Option {
	return func(publisher *fakePublisher) {
		publisher.faults = append(publisher.faults, outcome)
	}
}
