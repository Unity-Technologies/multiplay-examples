package game

import (
	"bytes"
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type MMProperties struct {
	MatchmakerEnabled bool
	BackfillEnabled   bool
}
type Team struct {
	TeamName  string
	TeamID    string
	PlayerIDs []string
}

type Player struct {
	Id         string        `json:"Id"`
	CustomData interface{}   `json:"CustomData"`
	QosResults []interface{} `json:"QosResults"`
}

type MatchProperties struct {
	Teams   []Team
	Players []Player
	Region  string
}

type AllocationPayload struct {
	MatchProperties  MatchProperties
	GeneratorName    string
	QueueName        string
	PoolName         string
	EnvironmentId    string
	BackfillTicketId string
}

type EncodedProperties struct {
	Data string
}

type BackfillTicket struct {
	queueName  string
	Attributes interface{}
	Properties EncodedProperties
	connection string
}

// Config structs

type MatchHosting struct {
	Type    string `json:"type"`
	Profile string `json:"profile"`
	FleetID string `json:"fleetId"`
}

type TeamConfig struct {
	Name      string `yaml:"Name"`
	TeamCount struct {
		Min int `yaml:"Min"`
		Max int `yaml:"Max"`
	} `yaml:"TeamCount"`
	PlayerCount struct {
		Min int `yaml:"Min"`
		Max int `yaml:"Max"`
	} `yaml:"PlayerCount"`
}

type MatchDefinition struct {
	Teams []TeamConfig `yaml:"Teams"`
}

type Rules struct {
	Name             string          `yaml:"Name"`
	DefaultQosRegion string          `yaml:"DefaultQosRegion"`
	BackfillEnabled  bool            `yaml:"BackfillEnabled"`
	MatchDefinition  MatchDefinition `yaml:"MatchDefinition"`
}

type MatchLogic struct {
	Type  string `json:"type"`
	Rules string `yaml:"rules"`
}

type Pool struct {
	Name         string        `json:"name"`
	Enabled      bool          `json:"enabled"`
	Default      bool          `json:"default"`
	Filters      []interface{} `json:"filters"`
	MatchHosting MatchHosting  `json:"matchHosting"`
	TimeoutMs    int           `json:"timeoutMs"`
	MatchLogic   MatchLogic    `json:"matchLogic"`
}

type Queue struct {
	Name                string `json:"name"`
	Enabled             bool   `json:"enabled"`
	Default             bool   `json:"default"`
	MaxPlayersPerTicket int    `json:"maxPlayersPerTicket"`
	Pools               []Pool `json:"Pools"`
}

type Config struct {
	Queues []Queue `json:"Queues"`
}

func (g *Game) GetMMMatchDef(queueName string, poolName string) MatchDefinition {
	g.logger.Infof("Getting match def for %s:%s from %+v", queueName, poolName, g.mmConfig.Queues)
	for _, queue := range g.mmConfig.Queues {
		if queue.Name == queueName {
			for _, pool := range queue.Pools {
				if pool.Name == poolName {
					g.logger.Infof("Got matchdef %+v", pool.MatchLogic)
					var rules Rules
					yaml.Unmarshal([]byte(pool.MatchLogic.Rules), &rules)
					g.logger.Infof("Unmarshalled %+v\n", rules)
					return rules.MatchDefinition
				}
			}
		}
	}
	g.logger.Infof("Could not get a matchdef")
	return MatchDefinition{}
}

func (g *Game) DecodeConfigJson(jsonConfig string) {
	g.logger.Infof("Decoding config")
	var config Config
	json.Unmarshal([]byte(jsonConfig), &config)
	g.mmConfig = config
	g.logger.Infof("Decoded config %+v", config)
}

func (g *Game) DecodeAllocationPayload(AllocationPayloadJson string) AllocationPayload {
	g.logger.Infof("Decoding: %s", AllocationPayloadJson)
	var allocationPayload AllocationPayload
	json.Unmarshal([]byte(AllocationPayloadJson), &allocationPayload)

	g.logger.Infof("Decoded: %+v", allocationPayload)
	return allocationPayload
}

func (g *Game) getAllocationPayload() {
	g.logger.Infof("Getting allocation payload")

	// Get allocation payload
	g.logger.Infof("calling Get on http://127.0.0.1:8086/payload/%s", g.backfillParams.AllocatedUUID)
	resp, err := http.Get("http://127.0.0.1:8086/payload/" + g.backfillParams.AllocatedUUID)
	if err != nil {
		g.logger.Infof("ERROR: %s", err.Error())
		g.logger.Fatalln(err)
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		g.logger.Infof("ERROR: %s", err.Error())
		g.logger.Fatalln(err)
	}

	//Convert the body to type string
	allocationPayloadJson := strings.Replace(string(body), "\\", "", -1)

	//g.logger.Info("Got response: " + allocationPayloadJson)
	//g.logger.Info("Got response!")

	result := g.DecodeAllocationPayload(allocationPayloadJson)
	g.logger.Infof("Decoded: %s", result)

	g.mmAllocationPayload = result

	g.logger.Infof("Got allocation payload %+v", result)
	//numberOfPlayer := 0
	//
	//g.logger.Info("Adding Players")
	//for _, team := range result.Teams {
	//	g.AddPlayers(int32(len(team.PlayerIDs)))
	//	numberOfPlayer += len(team.PlayerIDs)
	//}
	//g.logger.Info(strconv.Itoa(numberOfPlayer) + " players connected")
	//
	//g.logger.Info("Starting Daemon routine")
	//go MultiplayDaemon(g)
	//
	//logger.Info("Sleeping for 5 seconds")
	//time.Sleep(2 * time.Second)
	//logger.Info("Woke up and stopping server now")
	//_ = g.Stop()
}

func (g *Game) createBackfill(matchProperties MatchProperties) (*http.Response, error) {
	token, err := g.getJwtToken()
	if err != nil {
		g.logger.
			WithField("error", err.Error()).
			Error("Failed to get token from payload proxy.")

		return nil, err
	}

	backfillCreateURL := fmt.Sprintf("%s/v2/backfill/", g.backfillParams.MatchmakerURL)

	backfillData := BackfillTicket{}
	backfillData.queueName = ""
	backfillData.connection = g.gameBind.Addr().String() + string(g.port)

	matchPropertiesJson, err := json.Marshal(matchProperties)
	if err != nil {
		return nil, err
	}
	backfillData.Properties.Data = b64.StdEncoding.EncodeToString(matchPropertiesJson)

	backfillDataJson, err := json.Marshal(backfillData)
	g.logger.Infof("Creating backfill with: %s", backfillDataJson)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(context.Background(), "POST", backfillCreateURL, bytes.NewBuffer(backfillDataJson))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := g.httpClient.Do(req)

	return resp, err
}

func (g *Game) getMaxNumOfPlayers(matchDef MatchDefinition) int {
	g.logger.Infof("Getting max num of players from %+v", matchDef)
	totalPlayers := 0
	for i, team := range matchDef.Teams {
		g.logger.Infof("Team %d: %+v", i, matchDef)
		totalPlayers += team.TeamCount.Max * team.PlayerCount.Max
		g.logger.Infof("Total players %d", totalPlayers)
	}
	g.logger.Infof("Got max number of players %d", totalPlayers)
	return totalPlayers
}
func (g *Game) prepareMatchmaking() {
	configJson := `{
		"Queues": [
			{
			  "name": "default-queue",
			  "enabled": true,
			  "default": true,
			  "maxPlayersPerTicket": 4,
			  "Pools": [
				{
				  "name": "default-pool",
				  "enabled": true,
				  "default": true,
				  "filters": [],
				  "matchHosting": {
					"type": "Multiplay",
					"profile": "1078427",
					"fleetId": "7c2804ad-bc43-44bf-8b4e-469c9fa81a79"
				  },
				  "timeoutMs": 110000,
				  "matchLogic": {
					"type": "rules",
					"rules": "{Name: 'BasicRuleBasedConfig', DefaultQosRegion: 'bd984d6f-37a6-473d-a766-8944ae439526', BackfillEnabled: false, MatchDefinition: { Teams: [ { Name: 'Red Team', TeamCount: { Min: 1, Max: 1, }, PlayerCount: { Min: 1, Max: 10, } } ] }}"
				  }
				}
			  ]
			},
			{
			  "name": "golden-path-queue",
			  "enabled": true,
			  "default": false,
			  "maxPlayersPerTicket": 4,
			  "Pools": [
				{
				  "name": "default-pool",
				  "enabled": true,
				  "default": true,
				  "filters": [],
				  "matchHosting": {
					"type": "Multiplay",
					"profile": "1078427",
					"fleetId": "7c2804ad-bc43-44bf-8b4e-469c9fa81a79"
				  },
				  "timeoutMs": 110000,
				  "matchLogic": {
					"type": "rules",
					"rules": "{Name: 'BasicRuleBasedConfig', DefaultQosRegion: 'bd984d6f-37a6-473d-a766-8944ae439526', BackfillEnabled: false, MatchDefinition: { Teams: [ { Name: 'Red Team', TeamCount: { Min: 1, Max: 1, }, PlayerCount: { Min: 1, Max: 10, } } ] }}"
				  }
				}
			  ]
			},
			{
			  "name": "timeout-queue",
			  "enabled": true,
			  "default": false,
			  "maxPlayersPerTicket": 4,
			  "Pools": [
				{
				  "name": "default-pool",
				  "enabled": true,
				  "default": true,
				  "filters": [],
				  "matchHosting": {
					"type": "Multiplay",
					"profile": "1078427",
					"fleetId": "7c2804ad-bc43-44bf-8b4e-469c9fa81a79"
				  },
				  "timeoutMs": 100,
				  "matchLogic": {
					"type": "rules",
					"rules": "{Name: 'BasicRuleBasedConfig', DefaultQosRegion: 'bd984d6f-37a6-473d-a766-8944ae439526', BackfillEnabled: false, MatchDefinition: { Teams: [ { Name: 'Red Team', TeamCount: { Min: 1, Max: 1, }, PlayerCount: { Min: 1, Max: 10, } } ] }}"
				  }
				}
			  ]
			},
			{
			  "name": "incompatible-queue",
			  "enabled": true,
			  "default": false,
			  "maxPlayersPerTicket": 4,
			  "Pools": [
				{
				  "name": "default-pool",
				  "enabled": true,
				  "default": true,
				  "filters": [],
				  "matchHosting": {
					"type": "Multiplay",
					"profile": "1078427",
					"fleetId": "7c2804ad-bc43-44bf-8b4e-469c9fa81a79"
				  },
				  "timeoutMs": 60000,
				  "matchLogic": {
					"type": "rules",
					"rules": "{Name: 'BasicRuleBasedConfig', DefaultQosRegion: 'bd984d6f-37a6-473d-a766-8944ae439526', BackfillEnabled: false, MatchDefinition: { Teams: [ { Name: 'Red Team', TeamCount: { Min: 1, Max: 1, }, PlayerCount: { Min: 1, Max: 10, }, TeamRules: [ { Name: 'incompatible_rule', Type: 'Equality', Source: 'Players.CustomData.FieldNotPresentOnTicket', Reference: 'Value_Not_On_Ticket', Not: false, EnableRule: true}] } ] }}"
				  }
				}
			  ]
			},
			{
			  "name": "backfill-queue",
			  "enabled": true,
			  "default": false,
			  "maxPlayersPerTicket": 4,
			  "Pools": [
				{
				  "name": "default-pool",
				  "enabled": true,
				  "default": true,
				  "filters": [],
				  "matchHosting": {
					"type": "Multiplay",
					"profile": "1078427",
					"fleetId": "7c2804ad-bc43-44bf-8b4e-469c9fa81a79"
				  },
				  "timeoutMs": 60000,
				  "matchLogic": {
					"type": "rules",
					"rules": "{Name: 'BasicRuleBasedConfig', DefaultQosRegion: 'bd984d6f-37a6-473d-a766-8944ae439526', BackfillEnabled: true, MatchDefinition: { Teams: [ { Name: 'Red Team', TeamCount: { Min: 1, Max: 1, }, PlayerCount: { Min: 1, Max: 2, } } ] }}"
				  }
				}
			  ]
			},
			{
			  "name": "qos-queue",
			  "enabled": true,
			  "default": false,
			  "maxPlayersPerTicket": 4,
			  "Pools": [
				{
				  "name": "default-pool",
				  "enabled": true,
				  "default": true,
				  "filters": [],
				  "matchHosting": {
					"type": "Multiplay",
					"profile": "1078427",
					"fleetId": "7c2804ad-bc43-44bf-8b4e-469c9fa81a79"
				  },
				  "timeoutMs": 110000,
				  "matchLogic": {
					"type": "rules",
					"rules": "{ 'Name': 'BasicRuleBasedConfig', 'DefaultQosRegion': 'bd984d6f-37a6-473d-a766-8944ae439526', 'BackfillEnabled': false, 'MatchDefinition': { 'Teams': [ { 'Name': 'Main Team', 'TeamCount': { 'Min': 1, 'Max': 1 }, 'PlayerCount': { 'Min': 1, 'Max': 10 } } ], 'MatchRules': [ { 'Name': 'packet-loss-check', 'Type': 'LessThan', 'Source': 'Players.QosResults.PacketLoss', 'Reference': 0.2 } ] }}"
				  }
				}
			  ]
			},
			{
			  "name": "qos-queue-backfill",
			  "enabled": true,
			  "default": false,
			  "maxPlayersPerTicket": 4,
			  "Pools": [
				{
				  "name": "default-pool",
				  "enabled": true,
				  "default": true,
				  "filters": [],
				  "matchHosting": {
					"type": "Multiplay",
					"profile": "1078427",
					"fleetId": "7c2804ad-bc43-44bf-8b4e-469c9fa81a79"
				  },
				  "timeoutMs": 110000,
				  "matchLogic": {
					"type": "rules",
					"rules": "{ 'Name': 'BasicRuleBasedConfig', 'DefaultQosRegion': 'bd984d6f-37a6-473d-a766-8944ae439526', 'BackfillEnabled': true, 'MatchDefinition': { 'Teams': [ { 'Name': 'Main Team', 'TeamCount': { 'Min': 1, 'Max': 1 }, 'PlayerCount': { 'Min': 1, 'Max': 2 } } ], 'MatchRules': [ { 'Name': 'packet-loss-check', 'Type': 'LessThan', 'Source': 'Players.QosResults.PacketLoss', 'Reference': 0.2 } ] }}"
				  }
				}
			  ]
			}
	  	]
    }`
	g.DecodeConfigJson(configJson)
	g.getAllocationPayload()

	// If the MM created a backfill ticket use it to approve
	if g.mmAllocationPayload.BackfillTicketId != "" {
		g.backfillParams.BackfillTicketID = g.mmAllocationPayload.BackfillTicketId
	}

	currentConfig := g.GetMMMatchDef(g.mmAllocationPayload.QueueName, g.mmAllocationPayload.PoolName)
	maxPlayers := g.getMaxNumOfPlayers(currentConfig)
	g.state.MaxPlayers = int32(maxPlayers)

	g.logger.Infof("Match def: %s\nMax num of players: %s", currentConfig, maxPlayers)
}
func (g *Game) matchmakingThread() {
	ticker := time.NewTicker(1 * time.Second)

	for {
		if g.state.CurrentPlayers < g.state.MaxPlayers {
			if g.backfillParams.BackfillTicketID == "" {
				g.createBackfill(g.mmAllocationPayload.MatchProperties)
			}
		} else {
			if g.backfillParams.BackfillTicketID != "" {
				//	TODO: Delete the backfill ticket
			}
		}

		select {
		case <-ticker.C:
			if g.backfillParams.BackfillTicketID != "" {
				resp, err := g.approveBackfillTicket()
				if err != nil {
					g.logger.
						WithField("error", err.Error()).
						Error("encountered an error while in approve backfill loop.")
				} else {
					_ = resp.Body.Close()
					body, err := ioutil.ReadAll(resp.Body)
					g.logger.Infof("Approved backfill Ticket: %s\nError: %s", body, err.Error())
				}
			}
		case <-g.done:
			ticker.Stop()
			return
		}
	}
}

func (g *Game) keepAliveBackfill() {
	if g.backfillParams == nil {
		return
	}

	g.getAllocationPayload()
	if g.mmAllocationPayload.BackfillTicketId == "" {
		g.createBackfill(g.mmAllocationPayload.MatchProperties)
	}

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			resp, err := g.approveBackfillTicket()
			if err != nil {
				g.logger.
					WithField("error", err.Error()).
					Error("encountered an error while in approve backfill loop.")
			} else {
				_ = resp.Body.Close()
			}
		case <-g.done:
			ticker.Stop()

			return
		}
	}
}

// approveBackfillTicket is called in a loop to update and keep the backfill ticket alive.
func (g *Game) approveBackfillTicket() (*http.Response, error) {
	token, err := g.getJwtToken()
	if err != nil {
		g.logger.
			WithField("error", err.Error()).
			Error("Failed to get token from payload proxy.")

		return nil, err
	}

	resp, err := g.updateBackfillAllocation(token)
	if err != nil {
		g.logger.
			WithField("error", err.Error()).
			Errorf("Failed to update the matchmaker backfill allocations endpoint.")
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		err = errBackfillApprove
	}

	return resp, err
}

// getJwtToken calls the payload proxy token endpoint to retrieve the token used for matchmaker backfill approval.
func (g *Game) getJwtToken() (string, error) {
	payloadProxyTokenURL := fmt.Sprintf("%s/token?exp=18934560000", g.backfillParams.PayloadProxyURL)

	req, err := http.NewRequestWithContext(context.Background(), "GET", payloadProxyTokenURL, http.NoBody)
	if err != nil {
		return "", err
	}

	g.logger.Infof("Sending GET token request: %s", payloadProxyTokenURL)
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errTokenFetch
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tr tokenResponse
	err = json.Unmarshal(bodyBytes, &tr)

	if err != nil {
		return "", err
	}

	if len(tr.Error) != 0 {
		return "", errTokenFetch
	}

	g.logger.Infof("Got this token: %s", tr.Token)
	return tr.Token, nil
}

// updateBackfillAllocation calls the matchmaker backfill approval endpoint to update and keep the backfill ticket
// alive.
func (g *Game) updateBackfillAllocation(token string) (*http.Response, error) {
	if g.backfillParams.BackfillTicketID == "" {
		return nil, fmt.Errorf("no backfill ticket ID to approve")
	}

	backfillApprovalURL := fmt.Sprintf("%s/v2/backfill/%s/approvals",
		g.backfillParams.MatchmakerURL,
		g.backfillParams.BackfillTicketID)

	req, err := http.NewRequestWithContext(context.Background(), "POST", backfillApprovalURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	g.logger.Debugf("Sending POST backfill approval request: %s", backfillApprovalURL)
	resp, err := g.httpClient.Do(req)

	return resp, err
}
