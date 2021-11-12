package game

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/config"
	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/event"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func Test_watchConfig(t *testing.T) {
	l := logrus.NewEntry(logrus.New())
	p := path.Join(t.TempDir(), "config.json")

	require.NoError(t, ioutil.WriteFile(p, []byte(`{}`), 0600))

	g, err := New(l, p, 9000, 9001)
	require.NoError(t, err)
	require.NotNil(t, g)

	go g.processInternalEvents()
	<-g.internalEventProcessorReady

	// Allocate
	require.NoError(t, ioutil.WriteFile(p, []byte(`{
		"AllocationUUID": "alloc-uuid",
		"MaxPlayers": 12
	}`), 0600))
	require.Equal(t, event.Event{
		Type: event.Allocated,
		Config: &config.Config{
			AllocationUUID: "alloc-uuid",
			MaxPlayers:     12,
			QueryProtocol:  "sqp",
		},
	}, <-g.gameEvents)

	// Deallocate
	require.NoError(t, ioutil.WriteFile(p, []byte(`{
		"AllocationUUID": "",
		"MaxPlayers": 0
	}`), 0600))
	require.Equal(t, event.Event{
		Type: event.Deallocated,
		Config: &config.Config{
			QueryProtocol: "sqp",
		},
	}, <-g.gameEvents)

	close(g.done)
}
