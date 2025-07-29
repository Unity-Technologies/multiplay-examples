package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/internal/game"
	"github.com/sirupsen/logrus"
)

// parseFlags parses the supported flags and returns the values supplied to these flags.
func parseFlags(args []string) (string, string, string, error) {
	dir, _ := os.UserHomeDir()
	f := flag.NewFlagSet("simple-game-server-go", flag.ContinueOnError)

	var logTargets, logFile, tracebackLevel string
	f.StringVar(&logTargets, "log", "stdout,"+filepath.Join(dir, "logs"), "comma-separated log targets: 'stdout', file path, or directory")
	f.StringVar(&logFile, "logFile", "", "path to the log file to write to")
	f.StringVar(&tracebackLevel, "tracebackLevel", "none", "the amount of detail printed by the runtime prints before exiting due to an unrecovered panic")

	// Flags which are not used, but must be present to satisfy the default parameters in the Unity Dashboard.
	var port, queryPort uint
	f.UintVar(&port, "port", 8000, "port for the game server to bind to")
	f.UintVar(&queryPort, "queryport", 8001, "port for the query endpoint to bind to")

	return logTargets, logFile, tracebackLevel, f.Parse(args)
}

// logWritersFromTargets creates a multi-writer from the specified log targets and log file.
// If no valid targets are provided, it defaults to writing to stdout.
func logWritersFromTargets(logTargets string, logFile string, logger *logrus.Logger) io.Writer {
	targets := make([]io.Writer, 0)
	for _, t := range splitAndTrim(logTargets) {
		switch t {
		case "stdout":
			targets = append(targets, os.Stdout)
		case "stderr":
			targets = append(targets, os.Stderr)
		default:
			// If it's a directory, use server.log inside it
			info, err := os.Stat(t)
			if err == nil && info.IsDir() {
				t = filepath.Join(t, "server.log")
			}
			f, err := os.OpenFile(t, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
			if err != nil {
				logger.WithError(err).Warningf("could not open log target %s for writing", t)
				continue
			}
			targets = append(targets, f)
		}
	}
	// logFile takes precedence
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err == nil {
			targets = append(targets, f)
		} else {
			logger.WithError(err).Warning("could not open log file for writing")
		}
	}
	if len(targets) == 0 {
		return os.Stdout
	}
	return io.MultiWriter(targets...)
}

// splitAndTrim splits a string by the OS-specific path list separator and trims each part.
func splitAndTrim(s string) []string {
	parts := make([]string, 0)
	for _, p := range filepath.SplitList(s) {
		for _, t := range splitComma(p) {
			trimmed := filepath.Clean(t)
			if trimmed != "" && trimmed != "." {
				parts = append(parts, trimmed)
			}
		}
	}
	return parts
}

// splitComma splits a string by commas and trims each part, returning a slice of non-empty strings.
func splitComma(s string) []string {
	res := make([]string, 0)
	for _, t := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(t)
		if trimmed != "" && trimmed != "." {
			res = append(res, trimmed)
		}
	}
	return res
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	logTargets, logFile, tracebackLevel, err := parseFlags(os.Args[1:])
	if err != nil {
		logger.WithError(err).Fatal("error parsing flags")
	}

	if tracebackLevel != "" {
		logger.Infof("setting traceback level to %s", tracebackLevel)
		debug.SetTraceback(tracebackLevel)
	}

	logger.Out = logWritersFromTargets(logTargets, logFile, logger)

	g, err := game.New(logger)
	if err != nil {
		logger.WithError(err).Fatal("error creating game handler")
	}

	if err = g.Start(); err != nil {
		logger.WithError(err).Fatal("unable to start game")
	}
}
