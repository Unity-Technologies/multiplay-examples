package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the game server configuration.
type Config struct {
	// AllocatedUUID is the allocation ID provided to an event.
	AllocatedUUID string `json:"allocatedUUID"`

	// ReadBuffer is the size of the UDP connection read buffer.
	ReadBuffer int `json:"readBuffer,string"`

	// WriteBuffer is the size of the UDP connection write buffer.
	WriteBuffer int `json:"writeBuffer,string"`

	// MaxPlayers is the default value to report for max players.
	MaxPlayers uint32 `json:"maxPlayers,string"`

	// Map is the default value to report for map.
	Map string `json:"map"`

	// GameType is the default value to report for gametype.
	GameType string `json:"gameType"`

	// QueryType determines the protocol used for query responses.
	QueryType string `json:"queryType"`
}

// NewConfigFromFile loads configuration from the specified file
// and validates its contents.
func NewConfigFromFile(configFile string) (*Config, error) {
	var cfg *Config

	f, err := os.Open(configFile)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	defer f.Close()

	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	if cfg.ReadBuffer == 0 {
		cfg.ReadBuffer = 40960
	}

	if cfg.WriteBuffer == 0 {
		cfg.WriteBuffer = 40960
	}

	if cfg.MaxPlayers == 0 {
		cfg.MaxPlayers = 4
	}

	if cfg.Map == "" {
		cfg.Map = "Sample Map"
	}

	if cfg.GameType == "" {
		cfg.GameType = "Sample Game"
	}

	if cfg.QueryType == "" {
		cfg.QueryType = "sqp"
	}

	return cfg, nil
}
