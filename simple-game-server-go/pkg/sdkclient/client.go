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
	// SDK_DAEMON_URL represent sdk daemon default url
	SDK_DAEMON_URL = "localhost:8086"
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

// Subscribe creates subscription channel identified by server ID and subscribe to it.
func (s *SDKDaemonClient) Subscribe(channel string) error {
	sub, err := s.client.NewSubscription(channel)
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

	return err
}

// OnAllocate executes cb when an Allocate event is received from the server.
func (s *SDKDaemonClient) OnAllocate(cb AllocateCallback) {
	s.client.allocateFunc = cb
}

// OnDeallocate executes cb when a Deallocate event is received from the server.
func (s *SDKDaemonClient) OnDeallocate(cb DeallocateCallback) {
	s.client.deallocateFunc = cb
}

// ReadyForPlayers mark server as ready for players
func (s *SDKDaemonClient) ReadyForPlayers(serverID int64, allocationId string) error {
	url := fmt.Sprintf("http://%s"+ReadyForPlayersPath, s.url, serverID, allocationId)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("ready for players request: %s", err.Error())
	}

	statusCode, err := s.requestWithStatusCodeReturn(req)
	if err != nil {
		return fmt.Errorf("sdk request to daemon: %s", err.Error())
	}

	if statusCode != http.StatusOK {
		return fmt.Errorf("sdk request to daemon, unexpected status returned: %d", statusCode)
	}

	return nil
}

func (s *SDKDaemonClient) requestWithStatusCodeReturn(req *http.Request) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout)
	defer cancel()

	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return 0, err
	}

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
