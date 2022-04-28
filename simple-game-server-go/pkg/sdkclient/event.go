package sdkclient

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/twinj/uuid"
)

// EventType represents a type of server for a running server.
type EventType int

const (
	// AllocateEventType is dispatched when a server is allocated.
	AllocateEventType EventType = iota

	// DeallocateEventType is dispatched when a server is deallocated.
	DeallocateEventType
)

// MarshalJSON implements json.Marshaler
func (i EventType) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements json.Unmarshaler
func (i *EventType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch strings.ToLower(s) {
	case "allocateeventtype":
		*i = AllocateEventType
	case "deallocateeventtype":
		*i = DeallocateEventType
	default:
		return fmt.Errorf("unknown event type: %q", s)
	}

	return nil
}

// Event represents an server for a running game server.
type Event interface {
	// Type returns the type of server.
	Type() EventType
}

// BaseEvent represents fields common to all server.Event implementations.
type BaseEvent struct {
	EventID   string
	EventType EventType
	ServerID  int64
}

// Type returns the type of server.
func (e BaseEvent) Type() EventType {
	return e.EventType
}

func newBaseEvent(et EventType, serverID int64) *BaseEvent {
	return &BaseEvent{
		EventID:   uuid.NewV1().String(),
		EventType: et,
		ServerID:  serverID,
	}
}

// AllocateEvent is dispatched when a server is allocated.
type AllocateEvent struct {
	*BaseEvent

	AllocationID string
}

// NewAllocateEvent returns a new server allocate server.
func NewAllocateEvent(serverID int64, allocationID string) *AllocateEvent {
	return &AllocateEvent{
		BaseEvent:    newBaseEvent(AllocateEventType, serverID),
		AllocationID: allocationID,
	}
}

// DeallocateEvent is dispatched when a server is deallocated.
type DeallocateEvent struct {
	*BaseEvent

	AllocationID string
}

// NewDeallocateEvent returns a new server allocate server.
func NewDeallocateEvent(serverID int64, allocationID string) *DeallocateEvent {
	return &DeallocateEvent{
		BaseEvent:    newBaseEvent(DeallocateEventType, serverID),
		AllocationID: allocationID,
	}
}

// MarshalJSON implements json.Marshaler
func (a *AllocateEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(*a)
}

// MarshalJSON implements json.Marshaler
func (d *DeallocateEvent) MarshalJSON() ([]byte, error) {
	return json.Marshal(*d)
}

// UnmarshalEventJSON returns an Event unmarshaled from the JSON bytes.
func UnmarshalEventJSON(b []byte) (Event, error) {
	var be BaseEvent
	if err := json.Unmarshal(b, &be); err != nil {
		return nil, fmt.Errorf("unmarshal event from JSON: %w", err)
	}

	switch et := be.Type(); et {
	case AllocateEventType:
		var ae AllocateEvent
		if err := json.Unmarshal(b, &ae); err != nil {
			return nil, fmt.Errorf("unmarshal allocate event from JSON: %w", err)
		}

		return ae, nil
	case DeallocateEventType:
		var de DeallocateEvent
		if err := json.Unmarshal(b, &de); err != nil {
			return nil, fmt.Errorf("unmarshal deallocate event from JSON: %w", err)
		}

		return de, nil
	default:
		return nil, UnknownEventTypeError(et)
	}
}

// Compile-time assertion that the BaseEvent implements the expected Event
// methods.
var _ Event = BaseEvent{}
