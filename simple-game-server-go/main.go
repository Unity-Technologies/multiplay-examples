package main

import (
	"flag"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/internal/game"
	"github.com/sirupsen/logrus"
)

// parseFlags parses the supported flags and returns the values supplied to these flags.
func parseFlags(args []string) (string, string, string, error) {
	dir, _ := os.UserHomeDir()
	f := flag.NewFlagSet("simple-game-server-go", flag.ContinueOnError)

	var log, logFile, tracebackLevel string
	f.StringVar(&log, "log", filepath.Join(dir, "logs"), "path to the log directory to write to")
	f.StringVar(&logFile, "logFile", "", "path to the log file to write to")
	f.StringVar(
		&tracebackLevel,
		"tracebackLevel",
		"",
		"the amount of detail printed by the runtime prints before exiting due to an unrecovered panic",
	)

	// Flags which are not used, but must be present to satisfy the default parameters in the Unity Dashboard.
	var port, queryPort uint
	f.UintVar(&port, "port", 8000, "port for the game server to bind to")
	f.UintVar(&queryPort, "queryport", 8001, "port for the query endpoint to bind to")

	return log, logFile, tracebackLevel, f.Parse(args)
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	log, logFile, tracebackLevel, err := parseFlags(os.Args[1:])
	if err != nil {
		logger.WithError(err).Fatal("error parsing flags")
	}

	if tracebackLevel != "" {
		logger.Infof("setting traceback level to %s", tracebackLevel)
		debug.SetTraceback(tracebackLevel)
	}

	// Let -logFile take precedence over -log
	if logFile == "" && log != "" {
		logFile = filepath.Join(log, "server.log")
	}

	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
		if err == nil {
			defer f.Close()
			logger.Out = f
		} else {
			logger.WithError(err).Warning("could not open log file for writing")
		}
	}

	g, err := game.New(logger)
	if err != nil {
		logger.WithError(err).Fatal("error creating game handler")
	}

	if err = g.Start(); err != nil {
		logger.WithError(err).Fatal("unable to start game")
	}
}
