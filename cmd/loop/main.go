// Package main provides entry point for looop
package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/harmanloop/looop/nodeconn"
	"github.com/harmanloop/looop/protocol"
)

// Message represents wire format
type Message struct {
	Header  uint32
	Length  uint32
	Payload []byte
}

func (m *Message) validHeader() bool {
	return m.Header == protocol.Header
}

// ClientConn represents the connection between server and client
type ClientConn struct {
	net.Conn
	wg   *sync.WaitGroup
	in   chan *Message
	out  chan *Message
	done chan struct{}
}

func validHeader(rd io.Reader) bool {
	var buf [4]byte
	_, err := rd.Read(buf[:4])
	if err != nil {
		return false
	}
	hdr := binary.LittleEndian.Uint32(buf[:])
	return hdr != protocol.Header
}

const errNetClosing = "use of closed network connection"

var clients = newConnList()

func handleConnError(s *nodeconn.NodeConn) {
	s = clients.del(s)
	if s != nil {
		s.Stop()
	}
}

func main() {
	var wg sync.WaitGroup

	fmt.Println("looop is preparing to disrupt the industry!")
	addr, err := net.ResolveTCPAddr("tcp4", ":3377")
	if err != nil {
		log.Fatalf("Could not resolve: %v\n", err)
	}
	listener, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v\n", err)
	}

	listenerDone := make(chan error)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				listenerDone <- err
				break
			}
			c := nodeconn.NewWithErrHandler(conn, &wg, handleConnError)
			clients.put(c)
			wg.Add(2)
			go c.PollRead()
			go c.PollWrite()
			go func(c *nodeconn.NodeConn) {
				for {
					b, err := c.RawRead()
					if err != nil {
						continue
					}
					c.RawWrite(b)
				}
			}(c)
		}
	}()

	// Handle signals
	sigCaught := make(chan os.Signal)
	signal.Notify(sigCaught, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-sigCaught:
		fmt.Printf("Caught signal: %v\n", sig)
		if err := listener.Close(); err != nil {
			fmt.Println(err)
		}
		break
	case lerr := <-listenerDone:
		fmt.Println(lerr)
		break
	}
	clients.forEach(func(n *nodeconn.NodeConn) {
		delete(clients.m, n.RemoteAddr())
		n.Stop()
	})
	wg.Wait()
	fmt.Println("All done!")
}

type connList struct {
	sync.Mutex
	m map[net.Addr]*nodeconn.NodeConn
}

func (n *connList) put(conn *nodeconn.NodeConn) {
	n.Lock()
	n.m[conn.RemoteAddr()] = conn
	n.Unlock()
}

func (n *connList) del(conn *nodeconn.NodeConn) (old *nodeconn.NodeConn) {
	n.Lock()
	old, _ = n.m[conn.RemoteAddr()]
	delete(n.m, conn.RemoteAddr())
	n.Unlock()
	return
}

func (n *connList) forEach(fn func(s *nodeconn.NodeConn)) {
	n.Lock()
	for _, v := range clients.m {
		fn(v)
	}
	n.Unlock()
}

func newConnList() *connList {
	return &connList{m: make(map[net.Addr]*nodeconn.NodeConn)}
}
