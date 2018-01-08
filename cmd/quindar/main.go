package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/harmanloop/looop"
)

type ServerConn struct {
	net.Conn
	in   chan []byte
	out  chan []byte
	done chan struct{}
}

func (s *ServerConn) pollRead() {
	for {
		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil {
			fmt.Println(err)
		}
		select {
		case s.in <- buf[:n]:
		case <-s.done:
			goto Out
		}
	}
Out:
	fmt.Println("pollRead done!")
}

func (s *ServerConn) pollWrite() {
	for {
		select {
		case buf := <-s.out:
			fmt.Printf("s: %s, %v\n", string(buf), buf)
			_, err := s.Write(buf)
			if err != nil {
				fmt.Println(err)
				goto Out
			}
		case <-s.done:
			goto Out
		}
	}
Out:
	fmt.Println("pollWrite done!")
}

func (s *ServerConn) RawWrite(p []byte) {
	var pktLen [4]byte
	binary.LittleEndian.PutUint32(pktLen[:], uint32(len(p)))
	payload := append(pktLen[:], p...)
	buf := append(protocol.HeaderRaw, payload...)
	s.out <- buf
}

var (
	errUnknownHdr = errors.New("unknown protocol header")
	errShortHdr   = errors.New("short header")
)

func (s *ServerConn) RawRead() ([]byte, error) {
	buf := <-s.in
	// Drop empty messages
	if len(buf) <= len(protocol.HeaderRaw)+4 {
		return nil, errShortHdr
	}
	hdr := binary.LittleEndian.Uint32(buf[0:4])
	if hdr != protocol.Header {
		return nil, errUnknownHdr
	}
	length := binary.LittleEndian.Uint32(buf[4:8])
	return buf[8 : 8+length], nil
}

func main() {
	conn, err := net.Dial("tcp", ":3377")
	if err != nil {
		fmt.Println(err)
		return
	}
	srv := &ServerConn{
		Conn: conn,
		in:   make(chan []byte),
		out:  make(chan []byte),
		done: make(chan struct{}),
	}
	go srv.pollRead()
	go srv.pollWrite()

	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		s := stdin.Bytes()
		if len(s) == 0 {
			continue
		}
		b := make([]byte, len(s))
		copy(b, s)
		// srv.out <- b
		srv.RawWrite(b)
		// fmt.Println("read from srv:", string(<-srv.in))
		b, err := srv.RawRead()
		fmt.Printf("read from srv: %v, %v\n", b, err)
	}
}
