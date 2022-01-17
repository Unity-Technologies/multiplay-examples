package tcpmirror

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
)

type TCPMirror struct {
	l    net.Listener
	wg   sync.WaitGroup
	done chan struct{}
}

// New creates a new TCPMirror.
func New() TCPMirror {
	return TCPMirror{
		done: make(chan struct{}),
	}
}

// Start starts the tcp mirror.
func (t *TCPMirror) Start() (err error) {
	t.l, err = net.Listen("tcp", ":9200")
	if err != nil {
		return fmt.Errorf("net listen: %w", err)
	}

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		defer t.l.Close()
		for {
			if err := t.acceptConnection(t.l); err != nil {
				log.Println(fmt.Errorf("accept connection: %w", err))
				return
			}
		}
	}()
	return nil
}

// Stop stops the tcp mirror.
func (t *TCPMirror) Stop() {
	// By closing the connection we cause anything waiting on accept to instantly return and allow it to exit.
	t.l.Close()
	close(t.done)
	t.wg.Wait()
}

// acceptConnection accepts a single tcp connection and reflects any lines sent.
func (t *TCPMirror) acceptConnection(l net.Listener) error {
	conn, err := l.Accept()
	if err != nil {
		return fmt.Errorf("accept connection: %w", err)
	}
	// Spawn

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.handleConnection(conn)
	}()
	return nil
}

// handleConnection reflects and lines sent to it.
func (t *TCPMirror) handleConnection(conn net.Conn) {
	// Make a buffer to hold incoming data.
	reader := bufio.NewReader(conn)

	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Simulated game server received message from game client: %s", data)
		conn.Write([]byte(data))
	}
}
