package nodeconn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/harmanloop/looop/protocol"
)

type NodeConn struct {
	net.Conn
	wg   *sync.WaitGroup
	in   chan []byte
	out  chan []byte
	done chan struct{}
}

var (
	errUnknownHdr = errors.New("unknown protocol header")
	errShortHdr   = errors.New("short header")
)

func (s *NodeConn) PollRead() {
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
	s.wg.Done()
	fmt.Println("pollRead done!")
}

func (s *NodeConn) PollWrite() {
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
	s.wg.Done()
	fmt.Println("pollWrite done!")
}

func (s *NodeConn) Close() {
	close(s.done)
	s.Conn.Close()
}

func (s *NodeConn) RawWrite(p []byte) {
	var pktLen [4]byte
	binary.LittleEndian.PutUint32(pktLen[:], uint32(len(p)))
	payload := append(pktLen[:], p...)
	buf := append(protocol.HeaderRaw, payload...)
	s.out <- buf
}

func (s *NodeConn) RawRead() ([]byte, error) {
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

func New(conn net.Conn, wg *sync.WaitGroup) *NodeConn {
	if wg == nil {
		panic(fmt.Sprintf("nodeconn: wg is <nil>"))
	}
	return &NodeConn{
		Conn: conn,
		wg:   wg,
		in:   make(chan []byte),
		out:  make(chan []byte),
		done: make(chan struct{}),
	}
}
