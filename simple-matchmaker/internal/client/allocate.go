package mpclient

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// AllocateResponse contains the response from the api
type AllocateResponse struct {
	ProfileID int64
	UUID      string
	RegionID  string
	Created   string
	Error     string
}

type allocateResponseWrapper struct {
	Success    bool
	Allocation AllocateResponse
}

// Allocate allocates using the multiplay api
func (m *multiplayClient) Allocate(fleet, region string, profile int64, uuid string) (*AllocateResponse, error) {
	fmt.Println("Allocating")
	urlStr := fmt.Sprintf("%s/cfp/v2/fleet/%s/server/allocate", m.baseURL, fleet)
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("parse url %s", urlStr)
	}

	params := url.Values{}
	params.Add("regionid", region)
	params.Add("profileid", strconv.FormatInt(profile, 10))
	params.Add("uuid", uuid)
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("allocate new request")
	}

	req.SetBasicAuth(m.accessKey, m.secretKey)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send allocate request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("allocate call failed: %w", err)
	}

	var ar allocateResponseWrapper
	if err := json.NewDecoder(res.Body).Decode(&ar); err != nil {
		return nil, fmt.Errorf("decode allocate response: %w", err)
	}

	if !ar.Success {
		return nil, fmt.Errorf("allocation request failed")
	}

	return &ar.Allocation, nil
}
