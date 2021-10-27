package sqp

import (
	"github.com/Unity-Technologies/mp-game-server-sample-go/pkg/proto"
)

type (
	sqpServerInfo struct {
		CurrentPlayers uint16
		MaxPlayers     uint16
		ServerName     string
		GameType       string
		BuildID        string
		GameMap        string
		Port           uint16
	}
)

// queryStateToServerInfo converts proto.QueryState to sqpServerInfo.
func queryStateToServerInfo(qs *proto.QueryState) sqpServerInfo {
	if qs == nil {
		return sqpServerInfo{}
	}

	return sqpServerInfo{
		CurrentPlayers: uint16(qs.CurrentPlayers),
		MaxPlayers:     uint16(qs.MaxPlayers),
		ServerName:     qs.ServerName,
		GameType:       qs.GameType,
		GameMap:        qs.Map,
		Port:           qs.Port,
	}
}

// Size returns the number of bytes sqpServerInfo will use on the wire.
func (si sqpServerInfo) Size() uint32 {
	return uint32(
		2 + // CurrentPlayers
			2 + // MaxPlayers
			len([]byte(si.ServerName)) + 1 +
			len([]byte(si.GameType)) + 1 +
			len([]byte(si.BuildID)) + 1 +
			len([]byte(si.GameMap)) + 1 +
			2, // Port
	)
}
