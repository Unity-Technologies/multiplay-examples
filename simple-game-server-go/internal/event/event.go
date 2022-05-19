package event

import "github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/config"

type (
	// LifecycleEventType is a type of Multiplay game server lifecycle event.
	LifecycleEventType int

	// LifecycleEvent represents a Multiplay game server lifecycle event.
	LifecycleEvent struct {
		EventType      LifecycleEventType
		AllocationUUID string
		Config         *config.Config
	}
)

const (
	// Allocated indicates that a matchmaker has requested a game server from
	// Multiplay and this one has been chosen to host a match.
	Allocated = LifecycleEventType(iota)

	// Deallocated indicates that the matchmaker no longer requires this game
	// server to host a match.
	Deallocated
)
