package nodeconn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/harmanloop/looop/protocol"
)

// NodeConn represents a connection between server and client
type NodeConn struct {
	net.Conn
	wg          *sync.WaitGroup
	in          chan []byte
	out         chan []byte
	done        chan struct{}
	handleError func(*NodeConn)

	errMu sync.Mutex
	err   error
}

var (
	errUnknownHdr = errors.New("unknown protocol header")
	errShortHdr   = errors.New("short header")
)

// set error and return true if not set previously, otherwise return false
func (s *NodeConn) setError(err error) {
	var callHandler bool
	s.errMu.Lock()
	if s.err == nil {
		s.err = err
		callHandler = true
	}
	s.errMu.Unlock()

	if callHandler {
		s.handleError(s)
	}
}

func (s *NodeConn) Error() error {
	defer s.errMu.Unlock()
	s.errMu.Lock()
	return s.err
}

func (s *NodeConn) PollRead() {
	for {
		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil {
			s.setError(err)
			break
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
				s.handleError(s)
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

func (s *NodeConn) Stop() error {
	close(s.done)
	s.Conn.Close()
	return s.Error()
}

func (s *NodeConn) RawWrite(p []byte) {
	var header [8]byte
	copy(header[0:4], protocol.HeaderRaw)
	binary.LittleEndian.PutUint32(header[4:], uint32(len(p)))
	packet := append(header[:], p...)
	s.out <- packet
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
	return NewWithErrHandler(conn, wg, func(n *NodeConn) {})
}

func NewWithErrHandler(conn net.Conn, wg *sync.WaitGroup, eh func(*NodeConn)) *NodeConn {
	if wg == nil {
		panic(fmt.Sprint("nodeconn: wg is <nil>"))
	}
	return &NodeConn{
		Conn:        conn,
		wg:          wg,
		in:          make(chan []byte),
		out:         make(chan []byte),
		done:        make(chan struct{}),
		handleError: eh,
	}
}
