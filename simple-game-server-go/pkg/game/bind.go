package game

import (
	"net"
	"sync/atomic"
)

type (
	// udpBinding is a managed wrapper for a generic UDP listener.
	udpBinding struct {
		conn *net.UDPConn
		done int32
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
	}, nil
}

// Done marks the binding as complete, closing any open connections.
func (b *udpBinding) Done() {
	// TODO(dr): Use a chan struct{} instead of atomics?
	atomic.StoreInt32(&b.done, 1)

	if b.conn != nil {
		b.conn.Close()
		b.conn = nil
	}
}

// IsDone determines whether the binding is complete.
func (b udpBinding) IsDone() bool {
	return atomic.LoadInt32(&b.done) == 1
}
