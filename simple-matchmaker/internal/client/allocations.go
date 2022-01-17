package mpclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

var (
	// AllocationNotFound is returned when an allocation is not found.
	AllocationNotFound = errors.New("allocation not found")
)

// AllocationResponse is a response to query one or more allocations.
type AllocationResponse struct {
	ProfileID int64
	UUID      string
	Regions   string
	Created   string
	Requested string
	Fulfilled string
	ServerID  int64
	FleetID   string
	RegionID  string
	MachineID int64
	IP        string
	GamePort  int `json:"game_port"`
	Error     string
}

// allocationsResponseWrapper is a wrapper which contains the allocation api success status.
type allocationsResponseWrapper struct {
	Success     bool
	Allocations []AllocationResponse
}

// Allocations checks allocations using the multiplay api
func (m *multiplayClient) Allocations(fleet string, uuids ...string) ([]AllocationResponse, error) {
	urlStr := fmt.Sprintf("%s/cfp/v1/server/allocations", m.baseURL)
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parse url %s", urlStr)
	}

	params := url.Values{}
	params.Add("fleetid", fleet)
	for _, uuid := range uuids {
		params.Add("uuid", uuid)
	}
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("allocations new request")
	}

	req.Form = make(map[string][]string, 1)
	for _, uuid := range uuids {
		req.Form.Add("uuid", uuid)
	}

	req.SetBasicAuth(m.accessKey, m.secretKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send allocations request: %w", err)
	}
	defer res.Body.Close()

	switch {
	case res.StatusCode == http.StatusNotFound:
		return nil, AllocationNotFound
	case res.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("allocations call failed: %d: %s", res.StatusCode, getBody(res.Body))
	}

	var ar allocationsResponseWrapper
	if err := json.NewDecoder(res.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("decode allocations response: %w", err)
	}

	if !ar.Success {
		return nil, fmt.Errorf("allocations request failed")
	}

	return ar.Allocations, nil
}
