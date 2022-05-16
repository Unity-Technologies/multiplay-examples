package sdkclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/centrifugal/centrifuge-go"
	"github.com/sirupsen/logrus"
)

const (
	RequestTimeout      = 2 * time.Second
	ReadyForPlayersPath = "/v1/server/%d/allocation/%s/ready-for-players"
	SDKDaemonURL        = "http://localhost:8086"
)

// SDKDaemonClient provides a client for the SDK daemon.
//
// Callers MUST call the Errors() method to read any errors from the channel.
type SDKDaemonClient struct {
	client *centrifugeClientWrapper
	url    string
	logger *logrus.Entry
}

// NewSDKDaemonClient returns an SDK Daemon client configured to connect to the
// daemon on the given url.
func NewSDKDaemonClient(url string, l *logrus.Entry) *SDKDaemonClient {
	wsURL := fmt.Sprintf("ws://%s/v1/connection/websocket", url)

	sc := &SDKDaemonClient{
		client: &centrifugeClientWrapper{
			Client: centrifuge.NewJsonClient(wsURL, centrifuge.DefaultConfig()),
			errc:   make(chan error),
		},
		url:    url,
		logger: l,
	}

	return sc
}

// Connect subscribes connects to the SDK daemon and subscribes to events for this server
// identified by its process ID.
func (s *SDKDaemonClient) Connect() error {
	return s.client.Connect()
}

// Subscribe creates a subscription for the given server ID.
func (s *SDKDaemonClient) Subscribe(serverID int64) error {
	sub, err := s.client.NewSubscription(serverCentrifugeChannel(serverID))
	if err != nil {
		return fmt.Errorf("new subscription: %w", err)
	}

	s.logger.
		WithField("channel", sub.Channel()).
		WithField("NewSubscription", sub).
		Info("subscription created")

	sub.OnPublish(s.client)

	if err = sub.Subscribe(); err != nil {
		return fmt.Errorf("error subscribe: %w", err)
	}

	s.logger.
		WithField("channel", sub.Channel()).
		Info("subscribed")

	return nil
}

// OnAllocate executes cb when an Allocate event is received from the server.
func (s *SDKDaemonClient) OnAllocate(cb AllocateCallback) {
	s.client.allocateFunc = cb
}

// OnDeallocate executes cb when a Deallocate event is received from the server.
func (s *SDKDaemonClient) OnDeallocate(cb DeallocateCallback) {
	s.client.deallocateFunc = cb
}

// ReadyForPlayers mark server as ready for players.
func (s *SDKDaemonClient) ReadyForPlayers(serverID int64, allocationID string) error {
	url := fmt.Sprintf("http://%s"+ReadyForPlayersPath, s.url, serverID, allocationID)

	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("ready for players request: %w", err)
	}

	statusCode, err := s.requestWithStatusCodeReturn(req)
	if err != nil {
		return fmt.Errorf("sdk request to daemon: %w", err)
	}

	if statusCode != http.StatusOK {
		return UnexpectedHTTPStatusError(statusCode)
	}

	return nil
}

func (s *SDKDaemonClient) requestWithStatusCodeReturn(req *http.Request) (int, error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	return res.StatusCode, nil
}

// Errors returns a channel of underlying errors from the client.
//
// Callers MUST call this method to read from the channel.
func (s *SDKDaemonClient) Errors() <-chan error {
	return s.client.errc
}

// Close shuts down the underlying Centrifuge connection.
func (s *SDKDaemonClient) Close() error {
	return s.client.Close()
}

// serverCentrifugeChannel returns a Centrifuge channel name for the given server ID.
//
// We're using a Centrifuge user channel boundary here to ensure that only users
// identifying themselves as the given server can subscribe to this channel.
//
// See: https://centrifugal.dev/docs/server/channels#user-channel-boundary-
func serverCentrifugeChannel(serverID int64) string {
	return fmt.Sprintf("server#%d", serverID)
}
