package mpclient

import (
	"fmt"
	"net/http"
	"net/url"
)

func (m *multiplayClient) Deallocate(fleet, uuid string) error {
	fmt.Printf("deallocate: fid: %s uuid: %s\n", fleet, uuid)
	urlStr := fmt.Sprintf("%s/cfp/v2/fleet/%s/server/deallocate", m.baseURL, fleet)
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("parse url %s", urlStr)
	}

	params := url.Values{}
	params.Add("uuid", uuid)
	u.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return fmt.Errorf("deallocate new request")
	}

	req.SetBasicAuth(m.accessKey, m.secretKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send deallocate request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("deallocate call failed: %w", err)
	}

	return nil
}
