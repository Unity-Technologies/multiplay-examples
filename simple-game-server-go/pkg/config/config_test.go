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
					"AllocationUUID": "alloc-uuid",
					"QueryProtocol": "sqp"
				}`,
			},
			want: &Config{
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
			want: &Config{
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
