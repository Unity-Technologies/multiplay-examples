package game

import "net"

type (
	// udpBinding is a managed wrapper for a generic UDP listener.
	udpBinding struct {
		conn *net.UDPConn
		done chan struct{}
	}
)

// newUDPBinding creates a new UDP binding on the specified address.
func newUDPBinding(bindAddress string) (*udpBinding, error) {
	address, err := net.ResolveUDPAddr("udp4", bindAddress)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp4", address)
	if err != nil {
		return nil, err
	}

	return &udpBinding{
		conn: conn,
		done: make(chan struct{}),
	}, nil
}

// Close marks the binding as complete, closing any open connections.
func (b *udpBinding) Close() {
	if b.conn != nil {
		close(b.done)
		b.conn.Close()
		b.conn = nil
	}
}

// IsDone determines whether the binding is complete.
func (b udpBinding) IsDone() bool {
	select {
	case <-b.done:
		return true
	default:
		return false
	}
}
