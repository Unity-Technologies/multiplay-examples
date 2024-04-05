package game

import (
	"net"
)

// deallocated stops the currently running game, if one is running.
func (g *Game) deallocated() {
	close(g.alloc)
	g.disconnectAllClients()
	if err := g.gameBind.Close(); err != nil {
		g.logger.WithError(err).Error("error closing game")
	}

	g.gameBind = nil
	g.logger.Info("deallocated")
	g.logger = g.logger.WithField("allocation_uuid", "")
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
