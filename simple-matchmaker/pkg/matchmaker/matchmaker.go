package matchmaker

// PlayerInfo contains information about the players in the match.
type PlayerInfo struct {
	PlayerUUID string
	IP         string
}

// MatchInfo contains information about the match.
type MatchInfo struct {
	MatchedPlayers bool
	AllocationUUID string       `json:",omitempty"`
	Players        []PlayerInfo `json:",omitempty"`
	IP             string       `json:",omitempty"`
	Port           int          `json:",omitempty"`
	Aborted        bool         `json:",omitempty"`
}
