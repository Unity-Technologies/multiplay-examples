package sqp

import (
	"bytes"
	"testing"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/proto"
	"github.com/stretchr/testify/require"
)

func Test_Respond(t *testing.T) {
	t.Parallel()
	q, err := NewQueryResponder(&proto.QueryState{
		CurrentPlayers: 1,
		MaxPlayers:     2,
	})
	require.NoError(t, err)
	require.NotNil(t, q)

	addr := "client-addr:65534"

	// Challenge packet
	resp, err := q.Respond(addr, []byte{0, 0, 0, 0, 0})
	require.NoError(t, err)
	require.Equal(t, byte(0), resp[0])

	// Query packet
	resp, err = q.Respond(
		addr,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5], // challenge
				{0, 1},    // SQP version
				{1},       // Request chunks (server info only)
			},
			nil,
		),
	)
	require.NoError(t, err)
	require.Equal(
		t,
		bytes.Join(
			[][]byte{
				{1},
				resp[1:5],
				resp[5:7],
				{0},
				{0},
				{0x0, 0xe, 0x0, 0x0, 0x0, 0xa, 0x0, 0x1, 0x0, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			},
			nil,
		),
		resp,
	)
}
