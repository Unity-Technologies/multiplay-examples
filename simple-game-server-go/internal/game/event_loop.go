package game

import (
	"errors"
	"io"
	"path/filepath"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/config"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/event"
	"github.com/fsnotify/fsnotify"
)

// processInternalEvents processes internal events and watches the provided
// configuration file for changes.
// If changes are made, an allocation or deallocation event is fired depending
// on the state of AllocatedUUID.
func (g *Game) processInternalEvents() {
	w, _ := fsnotify.NewWatcher()
	_ = w.Add(filepath.Dir(g.cfgFile))

	g.wg.Add(1)
	g.internalEventProcessorReady <- struct{}{}
	defer g.wg.Done()

	for {
		select {
		case evt, ok := <-w.Events:
			if !ok {
				return
			}

			// Ignore events for other files.
			if evt.Name != g.cfgFile {
				continue
			}

			// We only care about when the config file has been rewritten.
			if evt.Op&fsnotify.Write != fsnotify.Write {
				continue
			}

			c, err := config.NewConfigFromFile(g.cfgFile)
			if err != nil {
				// Multiplay truncates the file when a deallocation occurs,
				// which results in two writes. The first write will produce an
				// empty file, meaning JSON parsing will fail.
				if !errors.Is(err, io.EOF) {
					g.logger.
						WithField("error", err.Error()).
						Error("error loading config")
				}

				continue
			}

			g.triggerAllocationEvents(c)

		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			g.logger.
				WithField("error", err.Error()).
				Error("error watching files")

		case <-g.done:
			_ = w.Close()
			close(g.gameEvents)
			close(g.internalEventProcessorReady)

			return
		}
	}
}

// triggerAllocationEvents triggers an allocation or deallocation event
// depending on the presence of an allocation ID.
func (g *Game) triggerAllocationEvents(c *config.Config) {
	if c.AllocatedUUID != "" {
		g.gameEvents <- event.Event{
			Type:   event.Allocated,
			Config: c,
		}
	} else {
		g.gameEvents <- event.Event{
			Type:   event.Deallocated,
			Config: c,
		}
	}
}
