package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Unity-Technologies/multiplay-examples/simple-matchmaker/pkg/matchmaker"
	"github.com/google/uuid"
)

var (
	//go:embed assets/help_en.txt
	helpEn string
)

func main() {
	showHelp := flag.Bool("help", false, "Display help")
	matchmakerURL := flag.String("matchmaker", "http://localhost:8085", "The URL where the sample matchmaker is running")
	flag.Parse()

	if *showHelp {
		displayHelp()
		return
	}

	fmt.Println("Starting to find a match")

	if err := matchmake(*matchmakerURL); err != nil {
		fmt.Println(fmt.Errorf("match: %w", err))
	}
	fmt.Println("Ending Match")
}

func matchmake(matchmakerURL string) (err error) {
	// Create a unique player id so the matchmaker can associate requests with us.
	playerID := uuid.New().String()

	matchInfo := &matchmaker.MatchInfo{}
	for {

		fmt.Printf("Asking matchmaker about match for us (playerid: %s)\n", playerID)

		// Repeatedly call the matchmakers player join endpoint
		matchInfo, err = requestPlayerJoin(playerID, matchmakerURL)
		if err != nil {
			return err
		}

		// There are three stages here in matchInfo.
		// MatchedPlayers is true - Matchmaker has found players to put together
		// AllocationUUID is non-empty - Matchmaker has requested allocation from api
		// IP address is non-empty - Matchmaker has been told the game is running here
		// We only care about the last one here.
		if matchInfo.IP != "" {
			// We got a match! Break out of the loop and play the match.
			fmt.Println("Matchmaker found us a match")
			break
		}

		fmt.Println("Matchmaker did not have a match ready")
		<-time.After(time.Second)
	}

	fmt.Printf("Connecting to match:\n")
	fmt.Printf(" - Allocation UUID: %s:%d\n", matchInfo.AllocationUUID)
	fmt.Printf(" - Address: %s:%d\n", matchInfo.IP, matchInfo.Port)
	fmt.Printf(" - Other players:\n")
	for _, pl := range matchInfo.Players {
		fmt.Printf(" - - %s - %s\n", pl.PlayerUUID, pl.IP)
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", matchInfo.IP, matchInfo.Port))
	if err != nil {
		return fmt.Errorf("resolve gameserver address: %w", err)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return fmt.Errorf("dial gameserver: %w", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, err = conn.Write([]byte(fmt.Sprintf("Player checking in: %s\n", playerID)))
			if err != nil {
				fmt.Printf("could not send to server: giving up: %s\n", err.Error())
				return
			}
			<-time.After(time.Second)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if err = conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				fmt.Printf("could not set read deadline for server: giving up: %s\n", err.Error())
			}
			content, err := ioutil.ReadAll(conn)
			if err != nil && !os.IsTimeout(err) {
				fmt.Printf("could not send to server: giving up: %s\n", err.Error())
				return
			}
			fmt.Print(string(content))
			<-time.After(time.Millisecond * 200)
		}
	}()
	wg.Wait()

	fmt.Println("Could not read or write to server. Match likely ended.")
	return nil
}

func requestPlayerJoin(playerID string, matchmakerURL string) (*matchmaker.MatchInfo, error) {
	player := matchmaker.PlayerInfo{
		PlayerUUID: playerID,
	}

	content, err := json.Marshal(player)
	if err != nil {
		return nil, fmt.Errorf("marshal player info: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, matchmakerURL+"/player", bytes.NewBuffer(content))
	if err != nil {
		return nil, fmt.Errorf("matchmaker player request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mathmaker send player request: %w", err)
	}

	matchInfo := matchmaker.MatchInfo{}
	err = json.NewDecoder(resp.Body).Decode(&matchInfo)
	if err != nil {
		return nil, fmt.Errorf("decode match info: %w", err)
	}

	return &matchInfo, err
}

func displayHelp() {
	fmt.Println(helpEn)
	fmt.Println("Arguments:")
	flag.PrintDefaults()
}
