package game

import (
	"errors"
	"fmt"
	"net"
	"time"
)

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

// Read reads data from the open connection into the supplied buffer.
func (b *udpBinding) Read(buf []byte) (int, *net.UDPAddr, error) {
	if b.IsDone() {
		return 0, nil, errors.New("binding is closed")
	}

	return b.conn.ReadFromUDP(buf)
}

// Write writes data to the specified UDP address.
func (b *udpBinding) Write(buf []byte, to *net.UDPAddr) (int, error) {
	if b.IsDone() {
		return 0, errors.New("binding is closed")
	}

	if err := b.conn.SetWriteDeadline(time.Now().Add(1 * time.Second)); err != nil {
		return 0, fmt.Errorf("error setting write deadline: %w", err)
	}

	return b.conn.WriteTo(buf, to)
}

// Close marks the binding as complete, closing any open connections.
func (b *udpBinding) Close() {
	if b.IsDone() {
		return
	}

	close(b.done)
	b.conn.Close()
	b.conn = nil
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
