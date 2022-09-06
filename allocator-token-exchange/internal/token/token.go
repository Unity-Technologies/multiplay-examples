package token

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const (
	tokenExchangeURL = "https://services.api.unity.com/auth/v1/token-exchange"
)

// Token is a unity token which refreshes.
type Token struct {
	projectID     string
	environmentID string
	accessKey     string
	secretKey     string

	token  string
	claims jwt.RegisteredClaims
}

// uggTokenExchange is the request body for the token exchange.
type uggTokenExchange struct {
	ApiKeyPublicIdentifier string   `json:"apiKeyPublicIdentifier"`
	Secret                 string   `json:"secret"`
	Scopes                 []string `json:"scopes"`
}

// uggTokenExchange is the returned body for the token exchange.
type uggAuthentication struct {
	AccessToken string `json:"accessToken"`
}

// NewToken creates a new token object which auto refreshes.
func NewToken(projectID, environmentID, accessKey, secretKey string) Token {
	return Token{
		projectID:     projectID,
		environmentID: environmentID,
		accessKey:     accessKey,
		secretKey:     secretKey,
	}
}

func (te *Token) RefreshToken() error {
	c := &http.Client{
		Timeout: time.Minute,
	}

	content, err := json.Marshal(&uggTokenExchange{
		ApiKeyPublicIdentifier: te.accessKey,
		Secret:                 te.secretKey,
		Scopes:                 []string{},
	})
	if err != nil {
		return fmt.Errorf("%w: marshal token exchange", err)
	}
	req, err := http.NewRequest(http.MethodPost, tokenExchangeURL, bytes.NewReader(content))
	q := req.URL.Query()
	q.Add("projectId", te.projectID)
	q.Add("environmentId", te.environmentID)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("%w: do token exchange", err)
	}
	defer func() {
		// We don't care about the body, just ensure it is drained and closed.
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("token exchange status code: %d", resp.StatusCode)
	}

	var o uggAuthentication
	err = json.NewDecoder(resp.Body).Decode(&o)
	if err != nil {
		return fmt.Errorf("%w: decode token exchange", err)
	}

	token, _, err := jwt.NewParser().ParseUnverified(o.AccessToken, &jwt.RegisteredClaims{})
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || claims == nil {
		return fmt.Errorf("claims invalid")
	}

	if claims.ExpiresAt == nil || claims.ExpiresAt.Unix() < time.Now().Unix() {
		return fmt.Errorf("received expired token")
	}

	te.claims = *claims
	te.token = o.AccessToken

	return nil
}

// BearerToken returns the bearer token
func (te *Token) BearerToken() (string, error) {
	if te.claims.ExpiresAt == nil || te.claims.ExpiresAt.Unix() < time.Now().Add(-time.Minute).Unix() {
		// We have no token, or it is about to expire, so refresh it.
		if err := te.RefreshToken(); err != nil {
			return "", fmt.Errorf("refresh token: %w", err)
		}
	}
	return te.token, nil
}

// AddRequestBearerToken adds the bearer token to a request
func (te *Token) AddRequestBearerToken(r *http.Request) error {
	t, err := te.BearerToken()
	if err != nil {
		return fmt.Errorf("get bearer token: %w", err)
	}
	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t))
	return nil
}
