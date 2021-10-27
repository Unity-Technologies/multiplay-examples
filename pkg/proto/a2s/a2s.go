package a2s

import (
	"bytes"
	"runtime"

	"github.com/Unity-Technologies/mp-game-server-sample-go/pkg/proto"
)

type (
	QueryResponder struct {
		enc   *encoder
		state *proto.QueryState
	}

	// infoWireFormat describes the format of a A2S_INFO query response.
	infoWireFormat struct {
		Header      []byte
		Protocol    byte
		ServerName  string
		GameMap     string
		GameFolder  string
		GameName    string
		SteamAppID  int16
		PlayerCount uint8
		MaxPlayers  uint8
		NumBots     uint8
		ServerType  byte
		Environment byte
		Visibility  byte
		VACEnabled  byte
	}
)

var (
	a2sInfoRequest  = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x54}
	a2sInfoResponse = []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x49}
)

// NewQueryResponder returns creates a new responder capable of responding
// to a2s-formatted queries.
func NewQueryResponder(state *proto.QueryState) (proto.QueryResponder, error) {
	q := &QueryResponder{
		enc:   &encoder{},
		state: state,
	}

	return q, nil
}

// Respond writes a query response to the requester in the SQP wire protocol.
func (q *QueryResponder) Respond(_ string, buf []byte) ([]byte, error) {
	if bytes.Equal(buf[0:5], a2sInfoRequest) {
		return q.handleInfoRequest()
	}

	return nil, NewErrUnsupportedQuery(buf[0:5])
}

func (q *QueryResponder) handleInfoRequest() ([]byte, error) {
	resp := bytes.NewBuffer(nil)
	f := infoWireFormat{
		Header:      a2sInfoResponse,
		Protocol:    1,
		ServerName:  "n/a",
		GameMap:     "n/a",
		GameFolder:  "n/a",
		GameName:    "n/a",
		Environment: environmentFromRuntime(runtime.GOOS),
	}

	if q.state != nil {
		f.ServerName = q.state.ServerName
		f.GameMap = q.state.Map
		f.PlayerCount = byte(q.state.CurrentPlayers)
		f.MaxPlayers = byte(q.state.MaxPlayers)
		f.GameName = q.state.GameType
	}

	if err := proto.WireWrite(resp, q.enc, f); err != nil {
		return nil, err
	}

	return resp.Bytes(), nil
}

func environmentFromRuntime(rt string) byte {
	switch rt {
	case "darwin":
		return byte('m')
	case "windows":
		return byte('w')
	default:
		return byte('l')
	}
}
