package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"

	mpclient "github.com/Unity-Technologies/multiplay-examples/simple-matchmaker/internal/client"
	"github.com/Unity-Technologies/multiplay-examples/simple-matchmaker/internal/simplematchmaker"
	"github.com/Unity-Technologies/multiplay-examples/simple-matchmaker/internal/simplematchmaker/tcpmirror"
)

var (
	//go:embed assets/help_en.txt
	helpEN string
)

func main() {
	showHelp := flag.Bool("help", false, "Display help")
	standalone := flag.Bool("standalone", false, "Mocks multiplay. Send clients to a game server mock running inside this matchmaker")
	fleetID := flag.String("fleet", "", "Fleet to use")
	regionID := flag.String("region", "", "Region to use")
	buildCfg := flag.Int64("buildcfg", 0, "Build configuration to use (Previously known as profile)")
	matchSize := flag.Int("matchsize", 2, "Size of matches to group players into")
	serverAddr := flag.String("server", ":8085", "Address to start server e.g. :8085, 192.168.1.100:8085")
	flag.Parse()

	if *showHelp {
		displayHelp()
		return
	}

	var backendClient mpclient.MultiplayClient
	cfg := simplematchmaker.Config{
		MatchSize: *matchSize,
	}

	if *standalone {
		backendClient = mpclient.MockMultiplayClient{}
	} else {
		if *fleetID == "" {
			displayArgs("No fleet specified")
			return
		}
		if *regionID == "" {
			displayArgs("No region specified")
			return
		}
		if *buildCfg == 0 {
			displayArgs("No buildcfg specified")
			return
		}

		cfg.FleetID = *fleetID
		cfg.RegionID = *regionID
		cfg.ProfileID = *buildCfg
		var err error
		backendClient, err = mpclient.NewClientFromEnv()
		if err != nil {
			log.Fatal(err)
		}
	}

	cfg.MatchSize = *matchSize
	mm := simplematchmaker.NewSimpleMatchmaker(cfg, backendClient)
	mm.Start()
	t := tcpmirror.New()
	if err := t.Start(); err != nil {
		panic(err)
	}

	http.HandleFunc("/player", mm.PlayerHandler)
	if err := http.ListenAndServe(*serverAddr, nil); err != nil {
		log.Println(err)
	}

	t.Stop()
	mm.Stop()
}

func displayHelp() {
	fmt.Println(helpEN)
	displayArgs()
}

func displayArgs(reasons ...string) {
	for _, reason := range reasons {
		fmt.Println(reason)
	}
	fmt.Println("Arguments:")
	flag.PrintDefaults()
}
