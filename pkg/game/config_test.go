package game

import (
	"io/ioutil"
	"path"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func Test_loadConfig(t *testing.T) {
	t.Parallel()
	type fields struct {
		configContent string
	}
	tests := []struct {
		name    string
		fields  fields
		want    *config
		wantErr bool
	}{
		{
			name: "loads config",
			fields: fields{
				configContent: `{
					"AllocationUUID": "alloc-uuid",
					"QueryProtocol": "sqp"
				}`,
			},
			want: &config{
				AllocationUUID: "alloc-uuid",
				QueryProtocol:  "sqp",
			},
		},
		{
			name: "defaults to sqp",
			fields: fields{
				configContent: `{
					"AllocationUUID": "alloc-uuid"
				}`,
			},
			want: &config{
				AllocationUUID: "alloc-uuid",
				QueryProtocol:  "sqp",
			},
		},
		{
			name: "malformed json",
			fields: fields{
				configContent: `bang!`,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := path.Join(t.TempDir(), "config.json")
			require.NoError(t, ioutil.WriteFile(f, []byte(tt.fields.configContent), 0600))

			got, err := loadConfig(f)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)

				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_watchConfig(t *testing.T) {
	l := logrus.NewEntry(logrus.New())
	p := path.Join(t.TempDir(), "config.json")

	require.NoError(t, ioutil.WriteFile(p, []byte(`{}`), 0600))

	g, err := New(l, p, 9000, 9001)
	require.NoError(t, err)
	require.NotNil(t, g)

	go g.processInternalEvents()
	<-g.internalEvents

	// Allocate
	require.NoError(t, ioutil.WriteFile(p, []byte(`{
		"AllocationUUID": "alloc-uuid",
		"MaxPlayers": 12
	}`), 0600))
	require.Equal(t, Event{
		Type: gameAllocated,
		Config: &config{
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
	require.Equal(t, Event{
		Type: gameDeallocated,
		Config: &config{
			QueryProtocol: "sqp",
		},
	}, <-g.gameEvents)

	require.NoError(t, g.Stop())
}
