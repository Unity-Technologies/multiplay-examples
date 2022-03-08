package game

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func (g *Game) keepAliveBackfill() {
	if g.backfillParams == nil {
		return
	}

	for {
		resp, err := g.approveBackfillTicket()
		if err != nil {
			g.logger.
				WithField("error", err.Error()).
				Error("encountered an error while in approve backfill loop.")
		} else {
			_ = resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
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

	return resp, err
}

// getJwtToken calls the payload proxy token endpoint to retrieve the token used for matchmaker backfill approval.
func (g *Game) getJwtToken() (string, error) {
	payloadProxyTokenURL := fmt.Sprintf("%s/token", g.backfillParams.PayloadProxyURL)

	req, err := http.NewRequestWithContext(context.Background(), "GET", payloadProxyTokenURL, http.NoBody)
	if err != nil {
		return "", err
	}

	g.logger.Debugf("Sending GET token request: %s", payloadProxyTokenURL)
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

	return tr.Token, nil
}

// updateBackfillAllocation calls the matchmaker backfill approval endpoint to update and keep the backfill ticket
// alive.
func (g *Game) updateBackfillAllocation(token string) (*http.Response, error) {
	backfillApprovalURL := fmt.Sprintf("%s/v2/backfill/%s/approvals",
		g.backfillParams.MatchmakerURL,
		g.backfillParams.AllocatedUUID)

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
