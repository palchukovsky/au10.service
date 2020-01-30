package au10

// Subscription represents subscription object.
type Subscription interface {
	// Close closes the subscription.
	Close()
	// GetErrChan returns channel errors or nil, of subscription closed by
	// cancel. Only one error is possible, after error channel will be closed.
	GetErrChan() <-chan error
}

type subscription struct {
	errChan chan error
}

func newSubscription() subscription {
	return subscription{errChan: make(chan error)}
}

func (subscr *subscription) close()                   { close(subscr.errChan) }
func (subscr *subscription) GetErrChan() <-chan error { return subscr.errChan }
