package game

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_BindLifecycle(t *testing.T) {
	t.Parallel()
	b, err := newUDPBinding(":0")
	require.NoError(t, err)
	require.NotNil(t, b)
	require.False(t, b.IsDone())

	endpoint, err := net.ResolveUDPAddr("udp4", b.conn.LocalAddr().String())
	require.NoError(t, err)
	client, err := net.DialUDP("udp4", nil, endpoint)
	require.NoError(t, err)

	expected := []byte("hello bind")
	_, err = client.Write(expected)
	require.NoError(t, err)

	actual := make([]byte, len(expected))
	_, _, err = b.Read(actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)

	b.Close()
	require.True(t, b.IsDone())
}
