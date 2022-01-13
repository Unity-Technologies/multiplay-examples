package game

import (
	"errors"
	"io"

	"github.com/Unity-Technologies/mp-sdk-daemon-poc/daemon/pkg/server"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/config"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/event"
)

// allocateHandler is an allocation event handler for the Multiplay SDK.
func (g *Game) allocateHandler(evt server.AllocateEvent) {
	// Reload the config file to pick up any changes made by the Multiplay
	// allocation process.
	c, err := config.NewConfigFromFile(g.cfgFile)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			g.logger.
				WithField("error", err.Error()).
				Error("error loading config")
		}

		return
	}

	g.gameEvents <- event.Event{
		Type:           event.Allocated,
		AllocationUUID: evt.AllocationID,
		Config:         c,
	}
}

// deallocateHandler is a deallocation event handler for the Multiplay SDK.
func (g *Game) deallocateHandler(evt server.DeallocateEvent) {
	// Reload the config file to pick up any changes made by the Multiplay
	// allocation process.
	c, err := config.NewConfigFromFile(g.cfgFile)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			g.logger.
				WithField("error", err.Error()).
				Error("error loading config")
		}

		return
	}

	g.gameEvents <- event.Event{
		Type:           event.Deallocated,
		AllocationUUID: evt.AllocationID,
		Config:         c,
	}
}
