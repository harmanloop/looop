// Package main provides entry point for looop
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

const version byte = 0x01

var (
	header = binary.LittleEndian.Uint32([]byte{'l', 'o', 'p', version})
)

// message represents wire format
type message struct {
	header  [4]byte
	length  uint32
	payload []byte
}

func (m *message) validHeader() bool {
	return binary.LittleEndian.Uint32(m.header[:]) == header
}

func (m *message) encode() {
}

func (m *message) decode() {
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	_, err := conn.Read(buf[:4])
	if err != nil {
		return
	}
	hdr := binary.LittleEndian.Uint32(buf[:4])
	if hdr != header {
		fmt.Println("Closing connection, unknown protocol header!", buf[:4])
		return
	}
	rd := bufio.NewReader(conn)
	for {
		n, err := rd.Read(buf)
		if err != nil && err == io.EOF {
			fmt.Println("Got EOF!")
			break
		}
		fmt.Printf("n: %d, %s, ra: %v\n", n, buf[:n], conn.RemoteAddr())
	}

	fmt.Printf("finished\n")
}

func main() {
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
		go handleConnection(conn)
	}
	// conn, err := listener.Accept()
	// conn.Write([]byte("What the fuck!\n"))
}
