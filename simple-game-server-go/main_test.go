package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func Test_parseFlags(t *testing.T) {
	t.Parallel()

	t.Run("single log target", func(t *testing.T) {
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
	})

	t.Run("multiple log targets", func(t *testing.T) {
		t.Parallel()
		log, logFile, tracebackLevel, err := parseFlags([]string{
			"-port", "9000",
			"-queryport", "9010",
			"-log", "stdout,/tmp/",
			"-logFile", "/tmp/Engine.log",
			"-tracebackLevel", "all",
		})

		require.NoError(t, err)
		require.Equal(t, "stdout,/tmp/", log)
		require.Equal(t, "/tmp/Engine.log", logFile)
		require.Equal(t, "all", tracebackLevel)
	})
}

func Test_logWritersFromTargets(t *testing.T) {
	t.Parallel()

	t.Run("stdout target", func(t *testing.T) {
		t.Parallel()
		logger := logrus.New()

		writer := logWritersFromTargets("stdout", "", logger)

		// Test that we can write to it (should not panic)
		n, err := writer.Write([]byte("test"))
		require.NoError(t, err)
		require.Greater(t, n, 0)
	})

	t.Run("stderr target", func(t *testing.T) {
		t.Parallel()
		logger := logrus.New()

		writer := logWritersFromTargets("stderr", "", logger)

		// Test that we can write to it (should not panic)
		n, err := writer.Write([]byte("test"))
		require.NoError(t, err)
		require.Greater(t, n, 0)
	})

	t.Run("file target", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.log")
		logger := logrus.New()

		writer := logWritersFromTargets(testFile, "", logger)

		// Write some data
		testData := "test log message\n"
		n, err := writer.Write([]byte(testData))
		require.NoError(t, err)
		require.Equal(t, len(testData), n)

		// Verify file was created and contains data
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("directory target", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "logs")
		require.NoError(t, os.MkdirAll(testDir, 0755))
		logger := logrus.New()

		writer := logWritersFromTargets(testDir, "", logger)

		// Write some data
		testData := "test log message in directory\n"
		n, err := writer.Write([]byte(testData))
		require.NoError(t, err)
		require.Equal(t, len(testData), n)

		// Verify server.log was created in the directory
		serverLogPath := filepath.Join(testDir, "server.log")
		content, err := os.ReadFile(serverLogPath)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("multiple targets", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		multiFile := filepath.Join(tempDir, "multi.log")
		targets := strings.Join([]string{"stdout", multiFile}, ",")
		logger := logrus.New()

		writer := logWritersFromTargets(targets, "", logger)

		// Write some data
		testData := "multi-target test\n"
		n, err := writer.Write([]byte(testData))
		require.NoError(t, err)
		require.Equal(t, len(testData), n)

		// Verify file was created
		content, err := os.ReadFile(multiFile)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("logFile takes precedence", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		priorityFile := filepath.Join(tempDir, "priority.log")
		logger := logrus.New()

		writer := logWritersFromTargets("stdout", priorityFile, logger)

		// Write some data
		testData := "priority file test\n"
		n, err := writer.Write([]byte(testData))
		require.NoError(t, err)
		require.Equal(t, len(testData), n)

		// Verify priority file was created
		content, err := os.ReadFile(priorityFile)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("invalid file target falls back to other targets", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		invalidPath := "/invalid/path/that/does/not/exist/file.log"
		validFile := filepath.Join(tempDir, "valid.log")
		targets := strings.Join([]string{invalidPath, validFile}, ",")
		logger := logrus.New()

		writer := logWritersFromTargets(targets, "", logger)

		// Write some data
		testData := "fallback test\n"
		n, err := writer.Write([]byte(testData))
		require.NoError(t, err)
		require.Equal(t, len(testData), n)

		// Verify only valid file was created
		_, err = os.Stat(invalidPath)
		require.True(t, os.IsNotExist(err))

		content, err := os.ReadFile(validFile)
		require.NoError(t, err)
		require.Equal(t, testData, string(content))
	})

	t.Run("no valid targets defaults to stdout", func(t *testing.T) {
		t.Parallel()
		invalidPath := "/invalid/path/that/does/not/exist"
		logger := logrus.New()

		writer := logWritersFromTargets(invalidPath, "", logger)

		// Should not be nil and should be writable
		require.NotNil(t, writer)
		n, err := writer.Write([]byte("test"))
		require.NoError(t, err)
		require.Greater(t, n, 0)
	})

	t.Run("empty targets defaults to stdout", func(t *testing.T) {
		t.Parallel()
		logger := logrus.New()

		writer := logWritersFromTargets("", "", logger)

		// Should not be nil and should be writable
		require.NotNil(t, writer)
		n, err := writer.Write([]byte("test"))
		require.NoError(t, err)
		require.Greater(t, n, 0)
	})
}

func Test_splitAndTrim(t *testing.T) {
	t.Parallel()

	t.Run("single target", func(t *testing.T) {
		t.Parallel()
		result := splitAndTrim("stdout")
		require.Equal(t, []string{"stdout"}, result)
	})

	t.Run("multiple comma-separated targets", func(t *testing.T) {
		t.Parallel()
		result := splitAndTrim("stdout,stderr,/tmp/test.log")
		require.Equal(t, []string{"stdout", "stderr", "/tmp/test.log"}, result)
	})

	t.Run("targets with whitespace", func(t *testing.T) {
		t.Parallel()
		result := splitAndTrim("stdout, stderr , /tmp/test.log ")
		require.Equal(t, []string{"stdout", "stderr", "/tmp/test.log"}, result)
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		result := splitAndTrim("")
		require.Empty(t, result)
	})

	t.Run("empty targets filtered out", func(t *testing.T) {
		t.Parallel()
		result := splitAndTrim("stdout,,stderr,")
		require.Equal(t, []string{"stdout", "stderr"}, result)
	})
}

func Test_splitComma(t *testing.T) {
	t.Parallel()

	t.Run("single value", func(t *testing.T) {
		t.Parallel()
		result := splitComma("stdout")
		require.Equal(t, []string{"stdout"}, result)
	})

	t.Run("multiple values", func(t *testing.T) {
		t.Parallel()
		result := splitComma("stdout,stderr,/tmp/log")
		require.Equal(t, []string{"stdout", "stderr", "/tmp/log"}, result)
	})

	t.Run("values with whitespace", func(t *testing.T) {
		t.Parallel()
		result := splitComma("stdout, stderr , /tmp/log ")
		require.Equal(t, []string{"stdout", "stderr", "/tmp/log"}, result)
	})

	t.Run("empty values filtered out", func(t *testing.T) {
		t.Parallel()
		result := splitComma("stdout,,stderr")
		require.Equal(t, []string{"stdout", "stderr"}, result)
	})
}
