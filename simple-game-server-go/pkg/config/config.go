package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Unity-Technologies/multiplay-examples/simple-game-server-go/pkg/sdkclient"
)

// Config represents the game server configuration.
type Config struct {
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

	// ServerID is the Multiplay game server ID.
	ServerID string `json:"serverID"`

	// SDKDaemonURL is the URL to the SDK daemon.
	SDKDaemonURL string `json:"sdkDaemonURL"`

	// MatchmakerURL is the public domain name used for approving backfill tickets.
	MatchmakerURL string `json:"matchmakerUrl"`

	// PayloadProxyURL is the url for the payload proxy which is used to retrieve the token.
	PayloadProxyURL string `json:"payloadProxyUrl"`

	// EnableBackfill enables backfill during the game loop.
	EnableBackfill string `json:"enableBackfill"`
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

	if cfg.SDKDaemonURL == "" {
		cfg.SDKDaemonURL = sdkclient.SDK_DAEMON_URL
	}

	if cfg.MatchmakerURL == "" {
		cfg.MatchmakerURL = "https://matchmaker.services.api.unity.com"
	}

	if cfg.PayloadProxyURL == "" {
		cfg.PayloadProxyURL = "http://localhost:8086"
	}

	if cfg.EnableBackfill == "" {
		cfg.EnableBackfill = "false"
	}

	return cfg, nil
}
