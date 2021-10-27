package game

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Unity-Technologies/mp-game-server-sample-go/pkg/proto"
	"github.com/Unity-Technologies/mp-game-server-sample-go/pkg/proto/a2s"
	"github.com/Unity-Technologies/mp-game-server-sample-go/pkg/proto/sqp"
)

type (
	EventType     = int
	InternalEvent = int

	Event struct {
		Type   EventType
		Config *config
	}
)

const (
	gameAllocated = EventType(iota)
	gameDeallocated
)

const (
	internalEventsProcessorReady = InternalEvent(iota)
	closeInternalEventsProcessor
)

// processEvents handles processing events for the operation of the
// game server, such as allocating and deallocating the server.
func (g *Game) processEvents() {
	g.wg.Add(1)
	defer g.wg.Done()

	for ev := range g.gameEvents {
		switch ev.Type {
		case gameAllocated:
			g.allocated(ev.Config)

		case gameDeallocated:
			g.deallocated(ev.Config)
		}
	}
}

// allocated starts a game after the server has been allocated.
func (g *Game) allocated(c *config) {
	g.logger = g.logger.WithField("allocation_uuid", c.AllocationUUID)
	port, _ := strconv.Atoi(c.Bind[1:])
	g.state = &proto.QueryState{
		MaxPlayers: int32(c.MaxPlayers),
		ServerName: fmt.Sprintf("r2 - %s", c.AllocationUUID),
		GameType:   "r2-demo-game",
		Map:        c.Map,
		Port:       uint16(port),
	}

	if err := g.switchQueryProtocol(*c); err != nil {
		g.logger.
			WithField("error", err.Error()).
			Error("error switching query protocol")

		return
	}

	go g.launchGame(*c)
}

// deallocated stops the currently running game, if one is running.
func (g *Game) deallocated(c *config) {
	g.disconnectAllClients()

	if g.gameBind != nil {
		_ = g.gameBind.Close()
		g.gameBind = nil
	}

	g.state = nil

	if c != nil {
		if err := g.switchQueryProtocol(*c); err != nil {
			g.logger.
				WithField("error", err.Error()).
				Error("error switching query protocol")
		}
	}

	g.logger.Info("deallocated")
	g.logger = g.logger.WithField("allocation_uuid", "")
}

// launchGame launches a TCP server which listens for connections. Data sent by clients
// is discarded.
func (g *Game) launchGame(c config) {
	g.logger.Info("allocated")
	addr, err := net.ResolveTCPAddr("tcp4", c.Bind)
	if err != nil {
		g.logger.
			WithField("error", err.Error()).
			Error("error resolving TCP address")

		return
	}

	gs, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		g.logger.
			WithField("error", err.Error()).
			Error("error listening on tcp port")

		return
	}

	g.gameBind = gs

	for {
		client, err := g.acceptClient(g.gameBind)
		if err != nil {
			if errors.Is(err, syscall.EINVAL) {
				g.logger.Debug("server closed")

				return
			}

			continue
		}

		go g.handleClient(client)
	}
}

// acceptClient accepts a new TCP connection and updates internal state.
func (g *Game) acceptClient(server *net.TCPListener) (*net.TCPConn, error) {
	client, err := server.AcceptTCP()
	if err != nil {
		return nil, err
	}

	if err := client.SetDeadline(time.Now().Add(1 * time.Minute)); err != nil {
		return nil, err
	}

	g.clients.Store(client.RemoteAddr(), client)
	atomic.AddInt32(&g.state.CurrentPlayers, 1)
	g.logger.Infof("connected: %s", client.RemoteAddr())

	return client, nil
}

// handleClient handles a interaction with one client
// connection.
func (g *Game) handleClient(client *net.TCPConn) {
	defer func() {
		g.clients.Delete(client.RemoteAddr())
		atomic.AddInt32(&g.state.CurrentPlayers, -1)
		g.logger.Infof("disconnected: %s", client.RemoteAddr())
	}()
	for {
		buf := make([]byte, 16)
		if _, err := client.Read(buf); err != nil {
			return
		}
	}
}

// disconnectAllClients disconnects all remaining clients connected to the game server.
func (g *Game) disconnectAllClients() {
	g.clients.Range(func(key interface{}, value interface{}) bool {
		client, ok := value.(*net.TCPConn)
		if !ok {
			return true
		}

		_ = client.Close()

		return true
	})
}

// switchQueryProtocol switches to a query protocol specified in the configuration.
// The query binding endpoints are restarted to serve on this endpoint.
func (g *Game) switchQueryProtocol(c config) error {
	var err error
	switch c.QueryProtocol {
	case "a2s":
		g.queryProto, err = a2s.NewQueryResponder(g.state)
	default:
		g.queryProto, err = sqp.NewQueryResponder(g.state)
	}

	if err != nil {
		return err
	}

	return g.restartQueryEndpoints(c)
}

// restartQueryEndpoints restarts the query endpoints onto the new endpoints specified in the configuration.
func (g *Game) restartQueryEndpoints(c config) error {
	for i := range g.queryBinds {
		g.queryBinds[i].Done()
	}

	g.queryBinds = make([]*udpBinding, len(c.BindQuery))
	for i, addr := range c.BindQuery {
		var err error
		if g.queryBinds[i], err = newUDPBinding(addr); err != nil {
			return err
		}
	}

	for i := range g.queryBinds {
		go handleQuery(
			g.queryProto,
			g.logger,
			&g.wg,
			g.queryBinds[i],
			c.ReadBuffer,
		)
	}

	return nil
}
