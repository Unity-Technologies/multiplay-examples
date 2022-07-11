package game

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/config"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/event"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/proto"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/proto/a2s"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/proto/sqp"
	"github.com/google/uuid"
)

type (
	// tokenResponse is the representation of a token and an error from the payload proxy service.
	tokenResponse struct {
		Token string `json:"token"`
		Error string `json:"error"`
	}
)

var (
	errTokenFetch      = errors.New("failed to retrieve JWT token")
	errBackfillApprove = errors.New("failed to approve backfill ticket")
)

// processEvents handles processing events for the operation of the
// game server, such as allocating and deallocating the server.
func (g *Game) processEvents() {
	g.wg.Add(1)
	defer g.wg.Done()

	for ev := range g.gameEvents {
		switch ev.Type {
		case event.Allocated:
			g.allocated(ev.Config)

		case event.Deallocated:
			g.deallocated(ev.Config)
		}
	}
}

// allocated starts a game after the server has been allocated.
func (g *Game) allocated(c *config.Config) {
	g.logger = g.logger.WithField("allocation_uuid", c.AllocatedUUID)
	g.logger.Infof("This is the config we got: %s", c)

	g.state = &proto.QueryState{
		MaxPlayers: int32(c.MaxPlayers),
		ServerName: fmt.Sprintf("simple-game-server-go - %s", c.AllocatedUUID),
		GameType:   c.GameType,
		Map:        c.Map,
		Port:       uint16(g.port),
	}
	mm, err := strconv.ParseBool(c.EnableMatchmaking)
	if err != nil {
		g.logger.
			WithField("error", err.Error()).
			Error("error parsing enableMatchmaker field in config")
	}

	g.mmProperties.MatchmakerEnabled = mm

	if g.mmProperties.MatchmakerEnabled {
		g.logger.Infof("Matchmaker is enabled, going to get backfill params")
		g.backfillParams = &proto.BackfillParams{
			MatchmakerURL:   c.MatchmakerURL,
			PayloadProxyURL: c.PayloadProxyURL,
			AllocatedUUID:   c.AllocatedUUID,
		}

		bf, err := strconv.ParseBool(c.EnableBackfill)
		if bf {
			g.mmProperties.BackfillEnabled = bf
		} else if err != nil {
			g.logger.
				WithField("error", err.Error()).
				Error("error parsing enableBackfill field in config")
		}
	}

	if err := g.switchQueryProtocol(*c); err != nil {
		g.logger.
			WithField("error", err.Error()).
			Error("error switching query protocol")

		return
	}

	go g.launchGame()
}

// deallocated stops the currently running game, if one is running.
func (g *Game) deallocated(c *config.Config) {
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
func (g *Game) launchGame() {
	g.logger.Info("allocated")
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%d", g.port))
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

	g.logger.Infof("game is: %s", g)
	g.logger.Infof("game is: %+v", g)
	if g.mmProperties.MatchmakerEnabled {
		g.logger.Infof("Matchmaker enabled")
		g.prepareMatchmaking()
	}

	ticker := time.NewTicker(1 * time.Second)
	timeSoFar := 0

	g.logger.Infof("Starting loop")
	for {
		if !g.mmProperties.MatchmakerEnabled {
			continue
		}

		g.logger.Infof("Going to check timer now")
		// Check matchmaker
		if g.mmProperties.BackfillEnabled {
			if g.state.CurrentPlayers < g.state.MaxPlayers {
				if g.backfillParams.BackfillTicketID == "" {
					g.createBackfill(g.mmAllocationPayload.MatchProperties)
				}
			} else {
				if g.backfillParams.BackfillTicketID != "" {
					//	TODO: Delete the backfill ticket
				}
			}
		}
		select {
		case <-ticker.C:
			timeSoFar += 1
			g.logger.Infof("Current time: %s", timeSoFar)

			if g.mmProperties.BackfillEnabled {
				if g.backfillParams.BackfillTicketID != "" {
					resp, err := g.approveBackfillTicket()
					if err != nil {
						g.logger.
							WithField("error", err.Error()).
							Error("encountered an error while in approve backfill loop.")
					} else {
						_ = resp.Body.Close()
						body, err := ioutil.ReadAll(resp.Body)
						g.logger.Infof("Approved backfill Ticket: %s\nError: %s", body, err.Error())
					}
				}
				if timeSoFar > 30 {
					ticker.Stop()
					g.Stop()
					return
				}
			} else {
				if timeSoFar > 3 {
					g.logger.Infof("Stopping server now because timer passed")
					ticker.Stop()
					g.Stop()
					return
				}
			}

		case <-g.done:
			ticker.Stop()
			g.Stop()
			return
		}

		go g.clientThread()
	}
}

func (g *Game) clientThread() {
	for {
		// Check player connection
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

	if err := client.SetDeadline(time.Now().Add(2 * time.Minute)); err != nil {
		return nil, err
	}

	g.clients.Store(client.RemoteAddr(), client)
	currentPlayers := atomic.AddInt32(&g.state.CurrentPlayers, 1)
	// TODO: We should have a map of clientID to player ID and add the player that connected
	// TODO: The client needs to send their ID when they connect, then we need to check that the DGS is expecting this player to join
	g.mmAllocationPayload.MatchProperties.Players = append(g.mmAllocationPayload.MatchProperties.Players, Player{Id: uuid.New().String()})
	g.logger.Infof("connected: %s, players: %d", client.RemoteAddr(), currentPlayers)

	return client, nil
}

// handleClient handles an interaction with one client connection.
func (g *Game) handleClient(client *net.TCPConn) {
	defer func() {
		g.clients.Delete(client.RemoteAddr())
		currentPlayers := atomic.AddInt32(&g.state.CurrentPlayers, -1)
		// TODO: We should have a map of clientID to player ID and remove the player that disconnected
		g.mmAllocationPayload.MatchProperties.Players = g.mmAllocationPayload.MatchProperties.Players[:len(g.mmAllocationPayload.MatchProperties.Players)-1]
		g.logger.Infof("disconnected: %s, players: %d", client.RemoteAddr(), currentPlayers)
	}()
	for {
		buf := make([]byte, 16)
		if _, err := client.Read(buf); err != nil {
			return
		}

		// Echo the packet back to the client, just to demonstrate that 2-way
		// communication is working.
		if _, err := client.Write(buf); err != nil {
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
func (g *Game) switchQueryProtocol(c config.Config) error {
	var err error
	switch c.QueryType {
	case "a2s":
		g.queryProto, err = a2s.NewQueryResponder(g.state)
	default:
		g.queryProto, err = sqp.NewQueryResponder(g.state)
	}

	if err != nil {
		return err
	}

	return g.restartQueryEndpoint(c)
}

// restartQueryEndpoint restarts the query endpoint to support a potential change of query protocol in the
// configuration.
func (g *Game) restartQueryEndpoint(c config.Config) error {
	if g.queryBind != nil {
		g.queryBind.Close()
	}

	var err error
	if g.queryBind, err = newUDPBinding(fmt.Sprintf(":%d", g.queryPort)); err != nil {
		return err
	}

	go handleQuery(
		g.queryProto,
		g.logger,
		&g.wg,
		g.queryBind,
		c.ReadBuffer,
	)

	return nil
}
