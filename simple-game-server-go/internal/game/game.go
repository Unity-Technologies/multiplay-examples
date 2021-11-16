package game

import (
	"net"
	"sync"
	"time"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/config"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/event"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/proto"
	"github.com/sirupsen/logrus"
)

type (
	// Game represents an instance of a game running on this server.
	Game struct {
		// cfgFile is the file path this game uses to read its configuration from
		cfgFile string

		// clients is a map of connected game clients:
		// - key:   string        - the remote IP of the client
		// - value: *net.TCPConn  - a connection object representing the client connection
		clients sync.Map

		// gameEvents is a channel of game events, for example allocated / deallocated
		gameEvents chan event.Event

		// gameBind is a TCP listener representing a fake game server
		gameBind *net.TCPListener

		// internalEventProcessorReady is a channel that, when written to,
		// indicates that the internal event processor is ready.
		internalEventProcessorReady chan struct{}

		// done is a channel that when closed indicates the server is going
		// away.
		done chan struct{}

		// logger handles structured logging for this game
		logger *logrus.Entry

		// port is the port number the game TCP server will listen on
		port uint

		// queryBind is a UDP endpoint which responds to game queries
		queryBind *udpBinding

		// queryPort is the port number the game query server will listen on
		queryPort uint

		// queryProto is an implementation of an interface which responds on a particular
		// query format, for example sqp, tf2e, etc.
		queryProto proto.QueryResponder

		// state represents current game states which are applicable to an incoming query,
		// for example current players, map name
		state *proto.QueryState

		// wg handles synchronising termination of all active
		// goroutines this game manages
		wg sync.WaitGroup
	}
)

// New creates a new game, configured with the provided configuration file.
func New(logger *logrus.Entry, configPath string, port, queryPort uint) (*Game, error) {
	g := &Game{
		cfgFile:                     configPath,
		gameEvents:                  make(chan event.Event, 1),
		logger:                      logger,
		internalEventProcessorReady: make(chan struct{}, 1),
		done:                        make(chan struct{}, 1),
		port:                        port,
		queryPort:                   queryPort,
	}

	return g, nil
}

// Start starts the game, opening the configured query and game ports.
func (g *Game) Start() error {
	c, err := config.NewConfigFromFile(g.cfgFile)
	if err != nil {
		return err
	}

	if err = g.switchQueryProtocol(*c); err != nil {
		return err
	}

	go g.processEvents()
	go g.processInternalEvents()

	// Wait until the internal event processor is ready.
	<-g.internalEventProcessorReady

	g.logger.
		WithField("port", g.port).
		WithField("queryport", g.queryPort).
		WithField("proto", c.QueryType).
		Info("server started")

	// Handle the app starting with an allocation
	if c.AllocatedUUID != "" {
		g.gameEvents <- event.Event{
			Type:   event.Allocated,
			Config: c,
		}
	}

	return nil
}

// Stop stops the game and closes all connections.
func (g *Game) Stop() error {
	g.logger.Info("stopping")

	if g.queryBind != nil {
		g.queryBind.Close()
	}

	g.gameEvents <- event.Event{Type: event.Deallocated}
	close(g.done)
	g.wg.Wait()
	g.logger.Info("stopped")

	return nil
}

// handleQuery handles responding to query commands on an incoming UDP port.
func handleQuery(q proto.QueryResponder, logger *logrus.Entry, wg *sync.WaitGroup, b *udpBinding, readBuffer int) {
	size := 16
	if readBuffer > 0 {
		size = readBuffer
	}

	wg.Add(1)
	defer wg.Done()

	for {
		buf := make([]byte, size)
		_, to, err := b.conn.ReadFromUDP(buf)
		if err != nil {
			if b.IsDone() {
				return
			}

			logger.
				WithField("error", err.Error()).
				Error("read from udp")

			continue
		}

		resp, err := q.Respond(to.String(), buf)
		if err != nil {
			logger.
				WithField("error", err.Error()).
				Error("error responding to query")

			continue
		}

		if err = b.conn.SetWriteDeadline(time.Now().Add(1 * time.Second)); err != nil {
			logger.
				WithField("error", err.Error()).
				Error("error setting write deadline")

			continue
		}

		if _, err = b.conn.WriteTo(resp, to); err != nil {
			logger.
				WithField("error", err.Error()).
				Error("error writing response")
		}
	}
}
