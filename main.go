package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/Unity-Technologies/mp-game-server-sample-go/pkg/game"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	config := ""
	log := ""
	f := flag.FlagSet{}
	f.StringVar(&config, "config", "", "path to the config file to use")
	f.StringVar(&log, "log", "", "path to the log file to write to")

	if err := f.Parse(os.Args[1:]); err != nil {
		logger.Fatal("msg", "error parsing flags", "err", err.Error())
	}

	if log != "" {
		logFile, err := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			defer logFile.Close()
			logger.Out = logFile
		} else {
			logger.Warningf("could not open log file for writing: %s", err.Error())
		}
	}

	g, err := game.New(logger.WithField("allocation_uuid", ""), config)
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
