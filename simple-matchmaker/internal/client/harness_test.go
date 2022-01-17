//go:build manual

package mpclient

import (
	"fmt"
	"testing"
	"time"

	"github.com/caarlos0/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type ManualTestConfig struct {
	FleetID     string `env:"FLEET"`
	RegionID    string `env:"REGION"`
	BuildConfig int64  `env:"BUILD_CFG"`
}

// TestMultiplayClient_Allocate is a basic test harness that runs through allocating, monitoring the allocation and
// deallocating. It needs the go build tag 'manual' setting to run. e.g. go test --tags manual
func TestMultiplayClient_Allocate(t *testing.T) {
	var cfg ManualTestConfig
	require.NoError(t, env.Parse(&cfg))

	c, err := NewClientFromEnv()
	require.NoError(t, err)

	allocUUID := uuid.New().String()

	fmt.Println("Making allocation")
	fmt.Println("Fleet: ", cfg.FleetID)
	fmt.Println("RegionID: ", cfg.RegionID)
	fmt.Println("BuildConfig(Profile): ", cfg.BuildConfig)
	fmt.Println("UUID: ", allocUUID)
	_, err = c.Allocate(cfg.FleetID, cfg.RegionID, cfg.BuildConfig, allocUUID)
	require.NoError(t, err)

	ticker := time.NewTicker(time.Second)

	fmt.Println("Waiting for allocation")
	for range ticker.C {
		allocs, err := c.Allocations(cfg.FleetID, allocUUID)
		require.NoError(t, err)

		if len(allocs) > 0 && allocs[0].IP != "" {
			fmt.Printf("Got allocation: %s:%d\n", allocs[0].IP, allocs[0].GamePort)
			break
		}
	}

	fmt.Println("Deallocating")
	require.NoError(t, c.Deallocate(cfg.FleetID, allocUUID))
	fmt.Println("Deallocated")
}
