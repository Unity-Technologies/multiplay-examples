package sdkclient

import "fmt"

// UnknownEventTypeError is returned in the event of an event with an unknown
// type being returned from the SDK daemon.
type UnknownEventTypeError EventType

func (e UnknownEventTypeError) Error() string {
	return fmt.Sprintf("unknown event type: %q", EventType(e).String())
}

// InvalidEventTypeError is returned when trying to parse an EventType from a
// string that does not match a valid value.
type InvalidEventTypeError string

func (e InvalidEventTypeError) Error() string {
	return fmt.Sprintf("invalid event type: %q", string(e))
}

// UnexpectedHTTPStatusError is returned when an HTTP request to the SDK daemon
// returns an unexpected status code.
type UnexpectedHTTPStatusError int

func (e UnexpectedHTTPStatusError) Error() string {
	return fmt.Sprintf("request to the SDK daemon returned an unexpected status code: %d", int(e))
}
