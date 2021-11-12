package event

import "github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/config"

type (
	// Typ is a type of Multiplay game server lifecycle event.
	Typ int

	// Event represents a Multiplay game server lifecycle event.
	Event struct {
		Type   Typ
		Config *config.Config
	}
)

const (
	// Allocated indicates that a matchmaker has requested a game server from
	// Multiplay and this one has been chosen to host a match.
	Allocated = Typ(iota)

	// Deallocated indicates that the matchmaker no longer requires this game
	// server to host a match.
	Deallocated
)
