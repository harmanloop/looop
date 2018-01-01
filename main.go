// Package main provides entry point for looop
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const version byte = 0x01

var (
	header = binary.LittleEndian.Uint32([]byte{'l', 'o', 'p', version})
)

// Message represents wire format
type Message struct {
	Header  uint32
	Length  uint32
	Payload []byte
}

func (m *Message) validHeader() bool {
	return m.Header == header
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
	return hdr != header
}

func handleRead(conn *ClientConn, wg *sync.WaitGroup, done <-chan struct{}) {
	rd := bufio.NewReader(conn)
L:
	for {
		buf := make([]byte, 512)
		n, err := rd.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Got EOF during read!")
			}
			break
		}
		fmt.Printf("n: %d, %s, ra: %v, err: %v\n", n, buf[:n], conn.RemoteAddr(), err)
		select {
		case conn.in <- &Message{Payload: buf[:n]}:
		case <-done:
			break L
		}
	}
	wg.Done()
}

func handleWrite(conn *ClientConn, wg *sync.WaitGroup, done <-chan struct{}) {
	var msg *Message
L:
	for {
		select {
		case msg = <-conn.out:
			n, err := conn.Write(msg.Payload)
			if err != nil {
				fmt.Println(err)
				break
			}
			fmt.Println("sndmsg:", n, err, string(msg.Payload))
		case <-done:
			break L
		}
	}
	wg.Done()
}

func handleConn(conn *ClientConn) {
	var wg sync.WaitGroup
	var done = make(chan struct{})

	wg.Add(2)
	go handleRead(conn, &wg, done)
	go handleWrite(conn, &wg, done)
	for {
		select {
		case msg := <-conn.in:
			fmt.Println("reading")
			fmt.Println(string(msg.Payload))
		case <-conn.done:
			goto out
		}
	}
out:
	close(done)
	conn.Close()
	wg.Wait()
	conn.wg.Done()
}

const errNetClosing = "use of closed network connection"

func main() {
	var wg sync.WaitGroup
	var clients = make([]*ClientConn, 0)

	fmt.Println("looop is preparing to disrupt the industry!")
	addr, err := net.ResolveTCPAddr("tcp4", ":3377")
	if err != nil {
		log.Fatalf("Could not resolve: %v\n", err)
	}
	listener, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Fatalf("Could not listen: %v\n", err)
	}

	listenerDone := make(chan error)
	go func() {
	L:
		for {
			conn, err := listener.Accept()
			if err != nil {
				listenerDone <- err
				break L
			}
			wg.Add(1)
			c := &ClientConn{
				Conn: conn,
				wg:   &wg,
				in:   make(chan *Message),
				out:  make(chan *Message),
				done: make(chan struct{}),
			}
			clients = append(clients, c)
			go handleConn(c)
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
		for _, cl := range clients {
			close(cl.done)
		}
	case lerr := <-listenerDone:
		fmt.Println(lerr)
	}
	wg.Wait()
	fmt.Println("All done!")
}
