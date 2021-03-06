package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseFlags(t *testing.T) {
	t.Parallel()
	config, log, port, queryPort, err := parseFlags([]string{
		"-config", "my-config.json",
		"-log", "/tmp/",
		"-port", "9000",
		"-queryport", "9001",
	})

	require.NoError(t, err)
	require.Equal(t, "my-config.json", config)
	require.Equal(t, "/tmp/", log)
	require.Equal(t, uint(9000), port)
	require.Equal(t, uint(9001), queryPort)
}
