package simplematchmaker

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	mpclient "github.com/Unity-Technologies/multiplay-examples/simple-matchmaker/internal/client"
	"github.com/Unity-Technologies/multiplay-examples/simple-matchmaker/pkg/matchmaker"
	"github.com/google/uuid"
)

var (
	// ErrNoMatch is returned when a match does not exist.
	ErrNoMatch = errors.New("no match found yet")

	// ErrMatchSearching is returned when a match is being searched for but not ready yet.
	ErrMatchSearching = errors.New("match searching")
)

// Config contains settings for starting the matchmaker
type Config struct {
	FleetID   string
	RegionID  string
	ProfileID int64
	MatchSize int
}

// SimpleMatchmaker defines a simple matchmaker.
type SimpleMatchmaker struct {
	// mpclient is library we use to interact with multiplay, or the standalone mock of multiplay.
	mpClient mpclient.MultiplayClient

	// unmatchedPlayers is a list of players which currently have not been matchmade yet.
	unmatchedPlayers []matchmaker.PlayerInfo
	// unmatchedPlayersMtx is a mutex to prevent races.
	unmatchedPlayersMtx sync.Mutex

	// matches is a map of match allocation UUIDs to matches.
	matches map[string]matchmaker.MatchInfo
	// matchesMtx is a mutex to prevent races.
	matchesMtx sync.Mutex

	// playerAlloc is a map of  player UUIDs to allocation UUIDs
	playerAlloc map[string]string
	// playerAllocsMtx is a mutex to prevent races.
	playerAllocsMtx sync.Mutex

	// done is a channel we use to tell any long running goroutines that we want to stop the matchmaker.
	done chan struct{}
	// wg is a waitgroup that ensures all running goroutines have ended before we allow a stop to complete.
	wg  sync.WaitGroup
	cfg Config
}

// NewSimpleMatchmaker creates a new simple matchmaker
func NewSimpleMatchmaker(cfg Config, client mpclient.MultiplayClient) *SimpleMatchmaker {
	return &SimpleMatchmaker{
		mpClient:    client,
		matches:     make(map[string]matchmaker.MatchInfo),
		playerAlloc: make(map[string]string),
		cfg:         cfg,
		done:        make(chan struct{}),
	}
}

// Start starts the simple matchmaker.
func (m *SimpleMatchmaker) Start() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-m.done:
				// We are stopping
				return
			case <-ticker.C:
				m.checkMatch()
			}
		}
	}()
}

// Stop stops the simple matchmaker, waiting for it to finish.
func (m *SimpleMatchmaker) Stop() {
	close(m.done)
	m.wg.Wait()
}

// PlayerHandler handles player calls to get a match.
func (m *SimpleMatchmaker) PlayerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Handle player")

	// Decode and validate the request
	var pl matchmaker.PlayerInfo
	if err := json.NewDecoder(r.Body).Decode(&pl); err != nil {
		fmt.Println("failed decoding request: " + err.Error())
		http.Error(w, "decode request", http.StatusBadRequest)
		return
	}
	pl.IP = r.RemoteAddr
	if pl.PlayerUUID == "" {
		fmt.Println("missing player uuid")
		http.Error(w, "missing player uuid", http.StatusBadRequest)
		return
	}

	// See if the player has a match that exists or is being searched for.
	matchInfo, err := m.playerMatch(pl.PlayerUUID)
	switch {
	case errors.Is(err, ErrMatchSearching):
		fmt.Printf("Player searching: %s\n", pl.PlayerUUID)
		// Match is currently being looked for. Nothing we can do right now so just return the current match info.
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(matchInfo)
		return
	case errors.Is(err, ErrNoMatch):
		m.queuePlayer(pl)
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Println("Adding new unmatched player")
	json.NewEncoder(w).Encode(matchInfo)
}

func (m *SimpleMatchmaker) queuePlayer(player matchmaker.PlayerInfo) {
	// We need a match so let's queue one.
	// Mark player as known but with no allocation.
	m.setPlayerAllocation(player.PlayerUUID, "")
	// This adds a player to our list of unmatched players. This will be processed in checkMatch when it ticks.
	m.addUnmatchedPlayer(player)
}

// checkMatch is called on an interval
func (m *SimpleMatchmaker) checkMatch() {
	// See how many matches we need to make.
	matchNum, playersWaiting := m.howManyMatchesNeeded()
	fmt.Printf("Players waiting: %d, Match size needed: %d\n", playersWaiting, m.cfg.MatchSize)

	// For each needed match, start the process of creating one.
	for i := 0; i < matchNum; i++ {
		// Find players from our unmatched waiting list
		players := m.grabPlayersForMatch()

		// Create a new match with a unique allocation UUID and the players in it.
		mi := matchmaker.MatchInfo{
			MatchedPlayers: true,
			Players:        players,
			AllocationUUID: uuid.New().String(),
		}

		// Update the player's info with details of this allocation.
		m.setAllocationForPlayers(players, mi.AllocationUUID)

		// Associate this match with the allocation UUID.
		m.setAllocationMatchInfo(mi.AllocationUUID, mi)

		// Allocate for this match.
		// NOTE: A real matchmaker may want to retry the allocate if it receives an error from this call, or if the
		// allocation has failed the following calls to monitor it.
		fmt.Println("About to allocate for match")
		if _, err := m.mpClient.Allocate(m.cfg.FleetID, m.cfg.RegionID, m.cfg.ProfileID, mi.AllocationUUID); err != nil {
			fmt.Printf("Failed to allocate %s\n", mi.AllocationUUID)
			m.matchCleanup(mi.AllocationUUID)
			continue
		}

		// Spawn a goroutine to wait for this allocation to be processed be multiplay.
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.waitForMatchToAllocate(mi.AllocationUUID)
		}()
	}
}

// waitForMatchToAllocate waits for a single allocation to allocate.
// NOTE: In a high throughput matchmaker you would need to supply multiple allocations to the 'allocations'
// endpoint (e.g. up to 100) to improve the performance.
func (m *SimpleMatchmaker) waitForMatchToAllocate(allocationUUID string) {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		allocs, err := m.mpClient.Allocations(m.cfg.FleetID, allocationUUID)
		switch {
		case errors.Is(err, mpclient.AllocationNotFound):
			// Allocation was not found. This could be because it was cancelled or the match started and ended quickly.
			m.matchCleanup(allocationUUID)
			fmt.Println("Allocation was not found (deallocated or started and ended quickly)")
			return
		}
		if err != nil {
			m.matchCleanup(allocationUUID)
			fmt.Printf("Non retryable issue for allocation: %s\n", allocationUUID)
			return
		}

		if len(allocs) == 0 {
			m.matchCleanup(allocationUUID)
			fmt.Println("Allocation was not found (deallocated or started and ended quickly)")
			return
		}

		alloc := allocs[0]
		if alloc.IP != "" {
			fmt.Printf("Got allocation: %s:%d\n", alloc.IP, alloc.GamePort)
			m.matchPlaying(allocationUUID, alloc)
			break
		}
		fmt.Printf("Waiting for allocation: %s\n", allocationUUID)
	}
}

func (m *SimpleMatchmaker) matchPlaying(allocationUUID string, alloc mpclient.AllocationResponse) {
	m.matchesMtx.Lock()
	defer m.matchesMtx.Unlock()
	v := m.matches[allocationUUID]
	v.IP = alloc.IP
	v.Port = alloc.GamePort
	m.matches[allocationUUID] = v
	m.scheduleMatchCleanup(allocationUUID)
}

// playerMatch gets the match a player is currently in.
func (m *SimpleMatchmaker) playerMatch(playerUUID string) (matchInfo *matchmaker.MatchInfo, err error) {
	alloc, ok := m.playerAlloc[playerUUID]
	if !ok {
		return nil, ErrNoMatch
	}
	if alloc == "" {
		// No match found for this player yet.
		fmt.Printf("Player %s asked about match status, but it was not ready.", playerUUID)
		return nil, ErrMatchSearching
	}

	mi, ok := m.matches[alloc]
	if !ok {
		// Should not have had a player alloc. Clean up and continue.
		delete(m.playerAlloc, playerUUID)
		return nil, ErrNoMatch
	}

	// Found a match that this player is in. Return it.
	return &mi, nil
}

// addUnmatchedPlayer adds an unmatched player into the unmatched player list.
func (m *SimpleMatchmaker) addUnmatchedPlayer(playerInfo matchmaker.PlayerInfo) {
	m.unmatchedPlayersMtx.Lock()
	defer m.unmatchedPlayersMtx.Unlock()
	m.unmatchedPlayers = append(m.unmatchedPlayers, playerInfo)
}

// setPlayerAllocation associates a player with an allocation
func (m *SimpleMatchmaker) setPlayerAllocation(playerUUID, allocationUUID string) {
	m.playerAllocsMtx.Lock()
	defer m.playerAllocsMtx.Unlock()
	m.playerAlloc[playerUUID] = allocationUUID
}

// howManyMatchesNeeded returns the number of matches needed to satisfy all queued players.
func (m *SimpleMatchmaker) howManyMatchesNeeded() (matchesNeeded, playersWaiting int) {
	m.unmatchedPlayersMtx.Lock()
	defer m.unmatchedPlayersMtx.Unlock()
	return len(m.unmatchedPlayers) / m.cfg.MatchSize, len(m.unmatchedPlayers)
}

// grabPlayersForMatch pulls players out of the unmatched player list and returns them.
func (m *SimpleMatchmaker) grabPlayersForMatch() []matchmaker.PlayerInfo {
	m.unmatchedPlayersMtx.Lock()
	defer m.unmatchedPlayersMtx.Unlock()
	matchPlayers := m.unmatchedPlayers[:m.cfg.MatchSize]
	m.unmatchedPlayers = m.unmatchedPlayers[m.cfg.MatchSize:]
	return matchPlayers
}

// setAllocationForPlayers sets all players allocation UUID
func (m *SimpleMatchmaker) setAllocationForPlayers(players []matchmaker.PlayerInfo, allocationUUID string) {
	m.playerAllocsMtx.Lock()
	defer m.playerAllocsMtx.Unlock()
	for _, p := range players {
		m.playerAlloc[p.PlayerUUID] = allocationUUID
	}
}

// setAllocationMatchInfo associates an allocation with a match.
func (m *SimpleMatchmaker) setAllocationMatchInfo(allocationUUID string, matchInfo matchmaker.MatchInfo) {
	m.matchesMtx.Lock()
	m.matchesMtx.Unlock()
	m.matches[allocationUUID] = matchInfo
}

// scheduleMatchCleanup removes the match after a delay. We want a delay so any players have time to get information
// about this match before we delete it.
func (m *SimpleMatchmaker) scheduleMatchCleanup(allocationUUID string) {
	go func() {
		select {
		case <-time.After(5 * time.Minute):
			m.matchCleanup(allocationUUID)
		case <-m.done:
			// The application is shutting down
			return
		}
	}()
}

// matchCleanup cleans up all the places we keep information about the match.
func (m *SimpleMatchmaker) matchCleanup(allocationUUID string) {
	m.matchesMtx.Lock()
	m.playerAllocsMtx.Lock()
	defer m.matchesMtx.Unlock()
	defer m.playerAllocsMtx.Unlock()

	delete(m.matches, allocationUUID)
	delete(m.playerAlloc, allocationUUID)
}
