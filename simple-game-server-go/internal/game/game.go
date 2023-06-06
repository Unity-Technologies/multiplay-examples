package game

import (
	"fmt"
	"net"
	"sync"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server"
	mserver "github.com/Unity-Technologies/unity-gaming-services-go-sdk/matchmaker/server"
	"github.com/sirupsen/logrus"
)

type (
	// Game represents an instance of a game running on this server.
	Game struct {
		*mserver.Server

		// clients is a map of connected game clients:
		// - key:   string        - the remote IP of the client
		// - value: *net.TCPConn  - a connection object representing the client connection
		clients sync.Map

		// done is a channel that when closed indicates the server is going
		// away.
		done chan struct{}

		// gameBind is a TCP listener representing a fake game server
		gameBind *net.TCPListener

		// logger handles structured logging for this game
		logger *logrus.Entry

		// wg handles synchronising termination of all active
		// goroutines this game manages
		wg sync.WaitGroup
	}
)

// New creates a new game, configured with the provided configuration file.
func New(logger *logrus.Logger) (*Game, error) {
	s, err := mserver.New(server.TypeAllocation)
	if err != nil {
		return nil, err
	}

	g := &Game{
		Server: s,
		logger: logger.WithField("allocation_uuid", ""),
		done:   make(chan struct{}),
	}

	return g, nil
}

// Start starts the game, opening the configured query and game ports.
func (g *Game) Start() error {
	g.logger.Info("starting")

	go g.processEvents()

	if err := g.Server.Start(); err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}

	g.logger.Info("started")

	defer func() {
		g.logger.Info("stopping")

		close(g.done)
		g.wg.Wait()

		g.logger.Info("stopped")
	}()

	return g.Server.WaitUntilTerminated()
}

// processEvents handles processing events for the operation of the
// game server, such as allocating and deallocating the server.
func (g *Game) processEvents() {
	g.wg.Add(1)
	defer g.wg.Done()

	for {
		select {
		case id := <-g.OnAllocate():
			g.allocated(id)

		case <-g.OnDeallocate():
			g.deallocated()

		case err := <-g.OnError():
			g.logger.WithError(err).Error("error maintaining server")

		case c := <-g.OnConfigurationChanged():
			g.logger.WithField("config", c).Info("configuration has changed")

		case <-g.done:
			return
		}
	}
}
