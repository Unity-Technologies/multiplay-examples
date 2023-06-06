package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseFlags(t *testing.T) {
	t.Parallel()
	log, logFile, tracebackLevel, err := parseFlags([]string{
		"-port", "9000",
		"-queryport", "9010",
		"-log", "/tmp/",
		"-logFile", "/tmp/Engine.log",
		"-tracebackLevel", "all",
	})

	require.NoError(t, err)
	require.Equal(t, "/tmp/", log)
	require.Equal(t, "/tmp/Engine.log", logFile)
	require.Equal(t, "all", tracebackLevel)
}
