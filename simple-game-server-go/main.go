package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/internal/game"
	"github.com/sirupsen/logrus"
)

// parseFlags parses the supported flags and returns the values supplied to these flags.
func parseFlags(args []string) (config string, log string, port uint, queryPort uint, err error) {
	var ip string
	dir, _ := os.UserHomeDir()
	f := flag.FlagSet{}

	f.StringVar(&config, "config", filepath.Join(dir, "server.json"), "path to the config file to use")
	f.StringVar(&log, "log", filepath.Join(dir, "logs"), "path to the log directory to write to")
	f.UintVar(&port, "port", 8000, "port for the event server to bind to")
	f.UintVar(&queryPort, "queryport", 8001, "port for the query endpoint to bind to")
	f.StringVar(&ip, "ip", "", "unused: required for full platform support")
	err = f.Parse(args)

	return
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	config, log, port, queryPort, err := parseFlags(os.Args[1:])
	if err != nil {
		logger.WithError(err).Fatal("error parsing flags")
	}

	if log != "" {
		logFile, err := os.OpenFile(filepath.Join(log, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			defer logFile.Close()
			logger.Out = logFile
		} else {
			logger.WithError(err).Warning("could not open log file for writing")
		}
	}

	g, err := game.New(logger.WithField("allocation_uuid", ""), config, port, queryPort)
	if err != nil {
		logger.WithError(err).Fatal("error creating game handler")
	}

	if err = g.Start(); err != nil {
		logger.WithError(err).Fatal("unable to start game")
	}

	// The Multiplay process management daemon will signal the event server to
	// stop. A graceful stop signal (SIGTERM) will be sent if the event server
	// fleet has been configured to support it.
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	_ = g.Stop()
}
