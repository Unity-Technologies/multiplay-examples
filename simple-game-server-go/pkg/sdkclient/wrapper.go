package sdkclient

import (
	"log"

	"github.com/centrifugal/centrifuge-go"
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

		errc chan error

		allocateFunc   AllocateCallback
		deallocateFunc DeallocateCallback
	}
)

// OnMessage implements centrifuge.MessageHandler.
func (c centrifugeClientWrapper) OnMessage(_ *centrifuge.Client, e centrifuge.MessageEvent) {
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
func (c centrifugeClientWrapper) OnPublish(_ *centrifuge.Subscription, e centrifuge.PublishEvent) {
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
