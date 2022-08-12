package game

import (
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/event"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func Test_watchConfig(t *testing.T) {
	l := logrus.NewEntry(logrus.New())
	p := path.Join(t.TempDir(), "config.json")

	require.NoError(t, os.WriteFile(p, []byte(`{}`), 0o600))

	g, err := New(l, p, 9000, 9001, &http.Client{Timeout: 1 * time.Second})
	require.NoError(t, err)
	require.NotNil(t, g)

	go g.processInternalEvents()
	<-g.internalEventProcessorReady

	// Allocate
	require.NoError(t, os.WriteFile(p, []byte(`{
		"allocatedUUID": "alloc-uuid",
		"maxPlayers": "12"
	}`), 0o600))
	ev := <-g.gameEvents
	require.Equal(t, event.Allocated, ev.Type)
	require.Equal(t, "alloc-uuid", ev.Config.AllocatedUUID)
	require.Equal(t, uint32(12), ev.Config.MaxPlayers)
	require.Equal(t, "sqp", ev.Config.QueryType)

	// Deallocate
	require.NoError(t, os.WriteFile(p, []byte(`{
		"allocatedUUID": ""
	}`), 0o600))
	ev = <-g.gameEvents
	require.Equal(t, event.Deallocated, ev.Type)
	require.Equal(t, "sqp", ev.Config.QueryType)
	require.Equal(t, "", ev.Config.AllocatedUUID)

	close(g.done)
}
