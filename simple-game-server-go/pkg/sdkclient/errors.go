package sdkclient

import "fmt"

// UnknownEventTypeError is returned in the event of an event with an unknown
// type being returned from the SDK daemon.
type UnknownEventTypeError EventType

func (e UnknownEventTypeError) Error() string {
	return fmt.Sprintf("unknown event type: %q", EventType(e).String())
}
