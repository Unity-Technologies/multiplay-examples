package config

import (
	"io/ioutil"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NewConfigFromFile(t *testing.T) {
	t.Parallel()
	type fields struct {
		configContent string
	}
	tests := []struct {
		name    string
		fields  fields
		want    *Config
		wantErr bool
	}{
		{
			name: "loads config",
			fields: fields{
				configContent: `{
					"allocatedUUID": "alloc-uuid",
					"readBuffer": "1024",
					"writeBuffer": "1024",
					"maxPlayers": "8",
					"map": "my-map",
					"gameType": "my-game",
					"queryType": "sqp",
					"matchmakerUrl": "https://matchmaker.services.api.unity.com",
					"payloadProxyUrl": "http://localhost:8086",
					"enableBackfill": "true"
				}`,
			},
			want: &Config{
				AllocatedUUID:   "alloc-uuid",
				ReadBuffer:      1024,
				WriteBuffer:     1024,
				MaxPlayers:      8,
				Map:             "my-map",
				GameType:        "my-game",
				QueryType:       "sqp",
				MatchmakerURL:   "https://matchmaker.services.api.unity.com",
				PayloadProxyURL: "http://localhost:8086",
				EnableBackfill:  "true",
			},
		},
		{
			name: "applies defaults",
			fields: fields{
				configContent: `{
					"allocatedUUID": "alloc-uuid"
				}`,
			},
			want: &Config{
				AllocatedUUID:   "alloc-uuid",
				ReadBuffer:      40960,
				WriteBuffer:     40960,
				MaxPlayers:      4,
				Map:             "Sample Map",
				GameType:        "Sample Game",
				QueryType:       "sqp",
				MatchmakerURL:   "https://matchmaker.services.api.unity.com",
				PayloadProxyURL: "http://localhost:8086",
				EnableBackfill:  "false",
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
			require.NoError(t, ioutil.WriteFile(f, []byte(tt.fields.configContent), 0o600))

			got, err := NewConfigFromFile(f)
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
