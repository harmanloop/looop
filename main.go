// Package main provides entry point for looop
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
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

func (m *Message) encode() {
}

func (m *Message) decode() {
}

// ClientConn represents the connection between server and client
type ClientConn struct {
	net.Conn
	ID  uint32
	in  chan *Message
	out chan *Message
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

func handleConnection(conn *ClientConn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	rd := bufio.NewReader(conn)
	for {
		n, err := rd.Read(buf)
		if err != nil && err == io.EOF {
			fmt.Println("Got EOF!")
			break
		}
		fmt.Printf("n: %d, %s, ra: %v, err: %v\n", n, buf[:n], conn.RemoteAddr(), err)
	}
	fmt.Printf("finished\n")
}

func handleRead(conn net.Conn, wg *sync.WaitGroup, done <-chan struct{}, in chan<- *Message) {
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
		case in <- &Message{Payload: buf[:n]}:
		case <-done:
			break L
		}
	}
	wg.Done()
}

func handleWrite(conn net.Conn, wg *sync.WaitGroup, done <-chan struct{}, out <-chan *Message) {
	var msg *Message
L:
	for {
		select {
		case msg = <-out:
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

func handleThisConn(conn net.Conn) {
	var wg sync.WaitGroup
	in := make(chan *Message)
	out := make(chan *Message)
	done := make(chan struct{})

	wg.Add(2)
	go handleRead(conn, &wg, done, in)
	go handleWrite(conn, &wg, done, out)
	for {
		select {
		case msg := <-in:
			fmt.Println("reading")
			fmt.Println(string(msg.Payload))
		case <-time.Tick(1 * time.Second):
			out <- &Message{Payload: []byte("fuck you\n")}
			fmt.Println("Wow sending a message")
			close(done)
			conn.Close()
			goto out
		}
	}
out:
	wg.Wait()
	fmt.Println("All done!")
}

func (c *ClientConn) startHandlers() error {
	go func() {
	}()
	return nil
}

type Clients struct {
	list []*ClientConn
}

func (c *Clients) add(cl *ClientConn) {
}

func (c *Clients) del(cl *ClientConn) {
}

func main() {
	// clients := make([]*ClientConn, 4)
	fmt.Println("looop is preparing to disrupt the industry!")

	addr, err := net.ResolveTCPAddr("tcp4", ":3377")
	if err != nil {
		log.Fatalf("Could not resolve: %v\n", err)
	}
	listener, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Fatalf("Could not listen: %v\n", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		// c := &ClientConn{Conn: conn, ID: 0}
		// clients = append(clients, c)
		// go handleConnection(c)
		go handleThisConn(conn)
	}
	// conn, err := listener.Accept()
	// conn.Write([]byte("What the fuck!\n"))
}
