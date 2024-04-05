package game

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"syscall"
	"time"

	"github.com/Unity-Technologies/unity-gaming-services-go-sdk/game-server-hosting/server"
	"github.com/sirupsen/logrus"
)

const (
	defaultMaxPlayers   = 4
	defaultReadyTimeout = 20 * time.Second
)

// allocated starts a game after the server has been allocated.
func (g *Game) allocated(allocationID string) {
	g.logger = g.logger.WithField("allocation_uuid", allocationID)
	g.alloc = make(chan struct{})

	c := g.Config()
	port, _ := c.Port.Int64()
	maxPlayers, _ := strconv.ParseInt(c.Extra["maxPlayers"], 10, 32)
	if maxPlayers == 0 {
		maxPlayers = defaultMaxPlayers
	}

	g.Server.SetMaxPlayers(int32(maxPlayers))
	g.Server.SetServerName(fmt.Sprintf("simple-game-server-go - %s", c.AllocatedUUID))
	g.Server.SetGameType(c.Extra["gameType"])
	g.Server.SetGameMap(c.Extra["map"])

	// Set a random metric, if using SQP.
	if c.QueryType == server.QueryProtocolSQP {
		if i, err := rand.Int(rand.Reader, big.NewInt(100)); err == nil {
			_ = g.SetMetric(0, float32(i.Int64()))
		}
	}

	go g.launchGame(port)
}

// launchGame launches a TCP server which listens for connections. Data sent by clients
// is discarded.
func (g *Game) launchGame(port int64) {
	g.logger.Info("allocated")
	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%d", port))
	if err != nil {
		g.logger.WithError(err).Error("error resolving TCP address")
		return
	}

	gs, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		g.logger.WithError(err).Error("error listening on TCP port")
		return
	}

	g.gameBind = gs

	go g.readyForPlayers()

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

	if err := client.SetDeadline(time.Now().Add(2 * time.Minute)); err != nil {
		return nil, err
	}

	g.clients.Store(client.RemoteAddr(), client)
	currentPlayers := g.Server.PlayerJoined()
	g.logger.WithFields(logrus.Fields{
		"client_ip":       client.RemoteAddr().String(),
		"current_players": currentPlayers,
	}).Info("client connected")

	return client, nil
}

// handleClient handles an interaction with one client connection.
func (g *Game) handleClient(client *net.TCPConn) {
	defer func() {
		g.clients.Delete(client.RemoteAddr())
		currentPlayers := g.Server.PlayerLeft()
		g.logger.WithFields(logrus.Fields{
			"client_ip":       client.RemoteAddr().String(),
			"current_players": currentPlayers,
		}).Info("client disconnected")
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

// readyForPlayers waits 20s then reports that the game is ready for players.
//
// This is to simulate a game server waiting for any initialization to complete
// before reporting that it is ready to accept players.
//
// You can optionally configure the timeout by setting the "readyTimeout"
// configuration variable in the build configuration to a duration string
// recognised by `time.ParseDuration`, e.g. "30s".
func (g *Game) readyForPlayers() {
	g.logger.Info("ready for players")

	timeout := defaultReadyTimeout

	if g.Config().Extra["readyTimeout"] != "" {
		t, err := time.ParseDuration(g.Config().Extra["readyTimeout"])
		if err != nil {
			g.logger.WithError(err).Error("parsing ready timeout")
		} else {
			timeout = t
		}
	}

	select {
	case <-time.After(timeout):
		g.logger.Info("ready timeout elapsed, reporting ready")
	case <-g.alloc:
		g.logger.Info("server no longer allocated, don't report ready")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := g.ReadyForPlayers(ctx); err != nil {
		g.logger.WithError(err).Error("reporting ready for players")
	}
}
