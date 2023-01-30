package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/internal/game"
	"github.com/sirupsen/logrus"
)

// parseFlags parses the supported flags and returns the values supplied to these flags.
func parseFlags(args []string) (config string, log string, logFile string, port uint, queryPort uint, tracebackLevel string, err error) {
	dir, _ := os.UserHomeDir()
	f := flag.FlagSet{}

	f.StringVar(&config, "config", filepath.Join(dir, "server.json"), "path to the config file to use")
	f.StringVar(&log, "log", filepath.Join(dir, "logs"), "path to the log directory to write to")
	f.StringVar(&logFile, "logFile", "", "path to the log file to write to")
	f.UintVar(&port, "port", 8000, "port for the game server to bind to")
	f.UintVar(&queryPort, "queryport", 8001, "port for the query endpoint to bind to")
	f.StringVar(&tracebackLevel, "tracebackLevel", "", "the amount of detail printed by the runtime prints before exiting due to an unrecovered panic")

	err = f.Parse(args)

	return
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	config, log, logFile, port, queryPort, tracebackLevel, err := parseFlags(os.Args[1:])
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

	g, err := game.New(
		logger.WithField("allocation_uuid", ""),
		config,
		port,
		queryPort,
		&http.Client{Timeout: time.Duration(1) * time.Second},
	)
	if err != nil {
		logger.WithError(err).Fatal("error creating game handler")
	}

	if err = g.Start(); err != nil {
		logger.WithError(err).Fatal("unable to start game")
	}

	// The Multiplay process management daemon will signal the game server to
	// stop. A graceful stop signal (SIGTERM) will be sent if the game server
	// fleet has been configured to support it.
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	_ = g.Stop()
}
