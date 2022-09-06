package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Unity-Technologies/multiplay-examples/allocator-token-exchange/internal/token"
)

// ExampleAuthenticatedCall tries to get a non-existent allocation. Authenticating correctly, but not finding it.
func ExampleAuthenticatedCall(token token.Token, projectID, environmentID, fleetID string) error {
	url := fmt.Sprintf(
		"https://services.api.unity.com/multiplay/v1/allocations/projects/%s/environments/%s/fleets/%s/allocations/00000000-0000-0000-0000-000000000000",
		projectID,
		environmentID,
		fleetID,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	err = token.AddRequestBearerToken(req)
	if err != nil {
		return fmt.Errorf("error adding bearer token: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer func() {
		// We don't care about the body, just ensure it is drained and closed.
		_, _ = io.Copy(ioutil.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("expecteed 404 but received: %d", resp.StatusCode)
	}
	return nil
}

func main() {
	var projectID, environmentId, fleetID, accessKey, secretKey string

	f := flag.FlagSet{}
	f.StringVar(&projectID, "projectID", "", "project id to use")
	f.StringVar(&environmentId, "environmentID", "", "environment id to use")
	f.StringVar(&fleetID, "fleetID", "", "environment id to use (optional for example call)")
	f.StringVar(&accessKey, "accessKey", "", "access key to use")
	f.StringVar(&secretKey, "secretKey", "", "secret key to use")
	if err := f.Parse(os.Args[1:]); err != nil {
		log.Fatal("error parsing flags", err.Error())
	}

	if environmentId == "" {
		log.Fatal("environmentID is required")
	}
	if projectID == "" {
		log.Fatal("projectID is required")
	}
	if fleetID == "" {
		fleetID = "39b2a8c1-b1f2-4083-a3fa-06f0f45724b8"
	}
	if accessKey == "" {
		log.Fatal("accessKey is required")
	}
	if secretKey == "" {
		log.Fatal("secretKey is required")
	}

	te := token.NewToken(projectID, environmentId, accessKey, secretKey)

	// Continuously try to get a token, and then make an authenticated call. Reuse token and refresh if needed.
	for {
		// You Example just getting the bearer token as a string.
		t, err := te.BearerToken()
		if err != nil {
			log.Println("failed to get bearer token", err)
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Println("Retrieved token", t[:25], "...", t[len(t)-25:])

		// Example of populating the Authorization header on a request.
		err = ExampleAuthenticatedCall(te, projectID, environmentId, fleetID)
		if err != nil {
			log.Println("failed to authenticate multiplay call", err)
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Println("Successfully authed call")
		time.Sleep(10 * time.Second)
	}
}
