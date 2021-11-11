package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/game"
	"github.com/sirupsen/logrus"
)

// parseFlags parses the supported flags and returns the values supplied to these flags.
func parseFlags(args []string) (config string, log string, port uint, queryPort uint, err error) {
	var ip string
	dir, _ := os.UserHomeDir()
	f := flag.FlagSet{}

	f.StringVar(&config, "config", filepath.Join(dir, "server.json"), "path to the config file to use")
	f.StringVar(&log, "log", filepath.Join(dir, "logs"), "path to the log directory to write to")
	f.UintVar(&port, "port", 8000, "port for the game server to bind to")
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
		logger.Fatal("msg", "error parsing flags", "err", err.Error())
	}

	if log != "" {
		logFile, err := os.OpenFile(filepath.Join(log, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			defer logFile.Close()
			logger.Out = logFile
		} else {
			logger.Warningf("could not open log file for writing: %s", err.Error())
		}
	}

	g, err := game.New(logger.WithField("allocation_uuid", ""), config, port, queryPort)
	if err != nil {
		logger.Fatal("msg", "error creating game handler", "err", err.Error())
	}

	if err = g.Start(); err != nil {
		logger.Fatal("msg", "unable to start game", "err", err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	_ = g.Stop()
}
