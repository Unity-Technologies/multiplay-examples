package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the game server configuration.
type Config struct {
	// AllocationUUID is the allocation ID provided to an event
	AllocationUUID string

	// ReadBuffer is the size of the UDP connection read buffer
	ReadBuffer int

	// WriteBuffer is the size of the UDP connection write buffer
	WriteBuffer int

	// MaxPlayers is the default value to report for max players.
	MaxPlayers uint32

	// Map is the default value to report for map.
	Map string

	// GameType is the default value to report for gametype.
	GameType string

	// QueryProtocol determines the protocol used for query responses
	QueryProtocol string
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

	if cfg.QueryProtocol == "" {
		cfg.QueryProtocol = "sqp"
	}

	return cfg, nil
}
