package sdkclient

import (
	"fmt"
	"log"
	"time"

	"github.com/centrifugal/centrifuge-go"
	"github.com/sirupsen/logrus"
)

// centrifugeClientWrapper exists to obfuscate the callback handlers of the
// Centrifuge client away from consumers of this package. Consumers should not
// be interacting with Centrifuge directly, instead using the abstraction
// provided by this package.
type (
	// AllocateCallback is a function called when an Allocate event is received
	// from the SDK daemon.
	AllocateCallback func(AllocateEvent)

	// DeallocateCallback is a function called when a Deallocate event is
	// received from the SDK daemon.
	DeallocateCallback func(DeallocateEvent)

	centrifugeClientWrapper struct {
		*centrifuge.Client

		sub *centrifuge.Subscription

		logger *logrus.Entry

		errc chan error
		done chan struct{}

		allocateFunc   AllocateCallback
		deallocateFunc DeallocateCallback
	}
)

// OnMessage implements centrifuge.MessageHandler.
func (c *centrifugeClientWrapper) OnMessage(_ *centrifuge.Client, e centrifuge.MessageEvent) {
	evt, err := UnmarshalEventJSON(e.Data)
	if err != nil {
		c.errc <- err
		return
	}

	log.Println("OnMessage: ", evt)
	switch evt.Type() {
	case AllocateEventType:
		if c.allocateFunc != nil {
			c.allocateFunc(evt.(AllocateEvent))
		}
	case DeallocateEventType:
		if c.deallocateFunc != nil {
			c.deallocateFunc(evt.(DeallocateEvent))
		}
	}
}

// OnPublish implements centrifuge.MessageHandler.
func (c *centrifugeClientWrapper) OnPublish(_ *centrifuge.Subscription, e centrifuge.PublishEvent) {
	evt, err := UnmarshalEventJSON(e.Data)
	if err != nil {
		c.errc <- err
		return
	}

	switch evt.Type() {
	case AllocateEventType:
		if c.allocateFunc != nil {
			c.allocateFunc(evt.(AllocateEvent))
		}
	case DeallocateEventType:
		if c.deallocateFunc != nil {
			c.deallocateFunc(evt.(DeallocateEvent))
		}
	}
}

// OnSubscribeError implements centrifuge.SubscribeErrorHandler.
func (c *centrifugeClientWrapper) OnSubscribeError(s *centrifuge.Subscription, e centrifuge.SubscribeErrorEvent) {
	c.logger.
		WithError(SubscribeError(e.Error)).
		WithField("channel", s.Channel()).
		Error("failed to subscribe")

	// Retry connecting to the SDK daemon. In some cases the server may be
	// attempting to connect before the SDK daemon has registered the existence
	// of the server.
	select {
	case <-c.done:
		return
	default:
		time.Sleep(1 * time.Second)

		if err := c.subscribe(); err != nil {
			c.logger.
				WithError(err).
				WithField("channel", s.Channel()).
				Error("failed to subscribe")
		}
	}
}

// OnSubscribeSuccess implements centrifuge.SubscribeSuccessHandler.
func (c *centrifugeClientWrapper) OnSubscribeSuccess(s *centrifuge.Subscription, _ centrifuge.SubscribeSuccessEvent) {
	c.logger.
		WithField("channel", s.Channel()).
		Info("subscribed to channel")
}

// Close stops any connection retries and closes the underlying client.
func (c *centrifugeClientWrapper) Close() error {
	close(c.done)

	return c.Client.Close()
}

// newSubscription wraps the underlying Centrifuge client methods to create a
// new subscription
func (c *centrifugeClientWrapper) newSubscription(channel string) error {
	var err error
	c.sub, err = c.Client.NewSubscription(channel)
	if err != nil {
		return fmt.Errorf("new subscription: %w", err)
	}
	c.sub.OnPublish(c)
	c.sub.OnSubscribeError(c)
	c.sub.OnSubscribeSuccess(c)

	return nil
}

// subscribe wraps the underlying Centrifuge client methods to subscribe to a
// channel.
func (c *centrifugeClientWrapper) subscribe() error {
	if err := c.sub.Subscribe(); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	return nil
}
