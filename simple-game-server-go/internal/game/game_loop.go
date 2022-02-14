package game

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/config"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/event"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/proto"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/proto/a2s"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/proto/sqp"
)

type (
	// tokenResponse is the representation of a token and an error from the payload proxy service.
	tokenResponse struct {
		Token string
		Error string
	}

	// tokenPayload represents the environment and project id of a token.
	tokenPayload struct {
		Upid string `json:"project_guid"`
		Env  string `json:"environment_id"`
	}
)

type errorWrapper struct {
	message string
}

func (e *errorWrapper) Error() string {
	return e.message
}

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
	g.state = &proto.QueryState{
		MaxPlayers: int32(c.MaxPlayers),
		ServerName: fmt.Sprintf("simple-game-server-go - %s", c.AllocatedUUID),
		GameType:   c.GameType,
		Map:        c.Map,
		Port:       uint16(g.port),
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

	go func() {
		c, err := g.getConfig()
		if err != nil {
			g.logger.
				WithField("error", err.Error()).
				Error("error loading config")

			return
		}
		bf, err := strconv.ParseBool(c.EnableBackfill)
		if !bf {
			if err != nil {
				g.logger.
					WithField("error", err.Error()).
					Error("error parsing enableBackfill field in config")
			}

			return
		}
		for {
			resp, err := g.approveBackfillTicket(c)
			if err != nil {
				g.logger.
					WithField("error", err.Error()).
					Error("encountered an error while in approve backfill loop.")
			} else {
				_ = resp.Body.Close()
			}
			time.Sleep(1 * time.Second)
		}
	}()

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

// approveBackfillTicket is called in a loop to update and keep the backfill ticket alive.
func (g *Game) approveBackfillTicket(c *config.Config) (*http.Response, error) {
	token, err := g.getJwtToken(c)
	if err != nil {
		g.logger.
			WithField("error", err.Error()).
			Error("Failed to get token from payload proxy.")

		return nil, err
	}

	resp, err := g.updateBackfillAllocation(c, token)
	if err != nil {
		g.logger.
			WithField("error", err.Error()).
			Errorf("Failed to update the matchmaker backfill allocations endpoint.")
	}

	return resp, err
}

// getJwtToken calls the payload proxy token endpoint to retrieve the token used for matchmaker backfill approval.
func (g *Game) getJwtToken(c *config.Config) (string, error) {
	payloadProxyTokenURL := fmt.Sprintf("%s/token", c.PayloadProxyURL)

	req, err := http.NewRequestWithContext(context.Background(), "GET", payloadProxyTokenURL, http.NoBody)
	if err != nil {
		return "", err
	}

	g.logger.Debugf("Sending GET token request: %s", payloadProxyTokenURL)
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &errorWrapper{resp.Status}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tr tokenResponse
	err = json.Unmarshal(bodyBytes, &tr)

	if err != nil {
		return "", err
	}

	if len(tr.Error) != 0 {
		err = &errorWrapper{tr.Error}

		return "", err
	}

	return tr.Token, nil
}

// updateBackfillAllocation calls the matchmaker backfill approval endpoint to update and keep the backfill ticket
// alive.
func (g *Game) updateBackfillAllocation(c *config.Config, token string) (*http.Response, error) {
	upid, env, err := g.parseJwtToken(token)
	if err != nil {
		return nil, err
	}

	backfillApprovalURL := fmt.Sprintf("%s/api/v2/%s/%s/backfill/%s/approvals",
		c.MatchmakerURL,
		upid,
		env,
		c.AllocatedUUID)

	req, err := http.NewRequestWithContext(context.Background(), "POST", backfillApprovalURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	g.logger.Debugf("Sending POST backfill approval request: %s", backfillApprovalURL)
	resp, err := g.httpClient.Do(req)

	return resp, err
}

// parseJwtToken extracts the project id and environment id from the JWT token.
func (g *Game) parseJwtToken(token string) (string, string, error) {
	payloadBytes, err := base64.RawStdEncoding.DecodeString(strings.Split(token, ".")[1])
	if err != nil {
		return "", "", err
	}

	var tp tokenPayload
	err = json.Unmarshal(payloadBytes, &tp)

	if err != nil {
		return "", "", err
	}

	return tp.Upid, tp.Env, nil
}

// getConfig loads the config from the configuration file into memory.
func (g *Game) getConfig() (*config.Config, error) {
	for {
		// Get Config from file
		c, err := config.NewConfigFromFile(g.cfgFile)
		if err != nil {
			// Multiplay truncates the file when a deallocation occurs,
			// which results in two writes. The first write will produce an
			// empty file, meaning JSON parsing will fail.
			if !errors.Is(err, io.EOF) {
				return nil, err
			}

			continue
		}

		return c, nil
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
	g.logger.Infof("connected: %s, players: %d", client.RemoteAddr(), currentPlayers)

	return client, nil
}

// handleClient handles an interaction with one client connection.
func (g *Game) handleClient(client *net.TCPConn) {
	defer func() {
		g.clients.Delete(client.RemoteAddr())
		currentPlayers := atomic.AddInt32(&g.state.CurrentPlayers, -1)
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
