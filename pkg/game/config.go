package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type (
	config struct {
		// AllocationUUID is the allocation ID provided to a game
		AllocationUUID string

		// Bind is the port to udpBinding to TCP game server to, e.g. "127.0.0.1:8080"
		Bind string

		// BindQuery is the address and port to bind to, e.g. "127.0.0.1:3075"
		BindQuery []string

		// ReadBuffer is the size of the UDP connection read buffer
		ReadBuffer int

		// WriteBuffer is the size of the UDP connection write buffer
		WriteBuffer int

		// MaxPlayers is the default value to report for max players.
		MaxPlayers uint32

		// Map is the default value to report for map.
		Map string

		// GameType is the default value to report for gametype.
		GameType string

		// QueryProtocol determines the protocol used for query responses
		QueryProtocol string
	}
)

var (
	ErrBindNotProvided      = errors.New("field Bind must be provided")
	ErrBindQueryNotProvided = errors.New("field BindQuery must be provided")
)

// processInternalEvents processes internal events and watches the provided
// configuration file for changes.
// If changes are made, an allocation or deallocation event is fired depending
// on the state of AllocationUUID.
func (g *Game) processInternalEvents() {
	w, _ := fsnotify.NewWatcher()
	_ = w.Add(filepath.Dir(g.cfgFile))

	g.wg.Add(1)
	g.internalEvents <- internalEventsProcessorReady
	defer g.wg.Done()

	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return
			}

			if event.Name != g.cfgFile {
				continue
			}

			// Config rewritten
			if event.Op&fsnotify.Write == fsnotify.Write {
				c, err := loadConfig(g.cfgFile)
				if err != nil {
					// Multiplay truncates the file when a deallocation occurs, which results in two writes.
					// The first write will produce an empty file, meaning JSON parsing will fail.
					if errors.Is(err, io.EOF) {
						continue
					}

					g.logger.
						WithField("error", err.Error()).
						Error("error loading config")

					continue
				}

				g.triggerAllocationEvents(c)
			}

		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			g.logger.
				WithField("error", err.Error()).
				Error("error watching files")

		case ev := <-g.internalEvents:
			switch ev {
			case closeInternalEventsProcessor:
				_ = w.Close()
				close(g.gameEvents)
				close(g.internalEvents)

				return

			case internalEventsProcessorReady:
				// re-queue event as we're not the one interested in this
				g.internalEvents <- ev
			}
		}
	}
}

// triggerAllocationEvents triggers an allocation or deallocation event depending on the presence of an allocation ID.
func (g *Game) triggerAllocationEvents(c *config) {
	if c.AllocationUUID != "" {
		g.gameEvents <- Event{
			Type:   gameAllocated,
			Config: c,
		}
	} else {
		g.gameEvents <- Event{
			Type:   gameDeallocated,
			Config: c,
		}
	}
}

// loadConfig loads configuration from the specified file
// and validates its contents.
func loadConfig(configFile string) (*config, error) {
	var cfg *config

	f, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	defer f.Close()

	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	if len(cfg.Bind) == 0 {
		return nil, ErrBindNotProvided
	}

	if len(cfg.BindQuery) == 0 {
		return nil, ErrBindQueryNotProvided
	}

	if cfg.QueryProtocol == "" {
		cfg.QueryProtocol = "sqp"
	}

	return cfg, nil
}
