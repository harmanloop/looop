package looop

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/harmanloop/looop/protocol"
)

// Server is a generic server to serve IPC requests over
// various transport layers
type Server struct {
	ls       net.Listener
	serverWg sync.WaitGroup
	quit     chan struct{}
	quitOnce sync.Once

	mu    sync.Mutex
	conns map[net.Addr]net.Conn

	opts options
}

// HandshakeHandler defines the function invoked to complete handshake after a
// new connection has been accepted. HandshakeHandler will return true if the
// server can speak the requested protocol and transport, and false otherwise.
type HandshakeHandler func(net.Conn, time.Duration) bool

type options struct {
	h                HandshakeHandler
	handshakeTimeout time.Duration
}

var defaultOptions = options{
	h:                defaultHandshake,
	handshakeTimeout: time.Second * 3,
}

// A Option sets various options for Server
type Option func(*options)

// WithHandshakeHandler lets you set handsake handler for server
func WithHandshakeHandler(h HandshakeHandler) Option {
	return func(o *options) {
		o.h = h
	}
}

// HandshakeTimeout lets you set the timeout for handshake
func HandshakeTimeout(t time.Duration) Option {
	return func(o *options) {
		o.handshakeTimeout = t
	}
}

func defaultHandshake(conn net.Conn, timeout time.Duration) bool {
	var done = make(chan struct{})
	var buf [protocol.HdrLen]byte
	var closeOnce sync.Once
	go func() {
		timer := time.NewTimer(timeout)
		select {
		case <-timer.C:
			closeOnce.Do(func() {
				conn.Close()
			})
		case <-done:
			timer.Stop()
		}
	}()
	_, err := io.ReadFull(conn, buf[:])
	close(done)
	if err != nil {
		goto errOut
	}
	if ok := protocol.ValidHdr(buf[:]); !ok {
		goto errOut
	}
	return true
errOut:
	closeOnce.Do(func() {
		conn.Close()
	})
	return false
}

func (s *Server) handleConn(conn net.Conn) {
}

// Serve accepts incoming connections on the listener ls
func (s *Server) Serve(ls net.Listener) error {
	defer s.serverWg.Done()
	s.serverWg.Add(1)
	s.ls = ls

	for {
		conn, err := s.ls.Accept()
		if err != nil {
			fmt.Println("failed to Accept:", err)
			select {
			case <-s.quit:
				return nil
			default:
			}
			return err
		}
		s.serverWg.Add(1)
		go func() {
			defer s.serverWg.Done()
			ok := s.opts.h(conn, s.opts.handshakeTimeout)
			if !ok {
				return
			}
			s.handleConn(conn)
		}()
	}
}

// New creates a new instance of Server
func New(opt ...Option) *Server {
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}
	return &Server{
		quit:  make(chan struct{}),
		conns: make(map[net.Addr]net.Conn),
		opts:  opts,
	}
}

// Stop stops the server, closes all connections
func (s *Server) Stop() {
	s.quitOnce.Do(func() {
		close(s.quit)
		s.ls.Close()
	})

	s.mu.Lock()
	if s.conns == nil {
		s.mu.Unlock()
		return
	}
	cs := s.conns
	s.conns = nil
	s.mu.Unlock()

	for _, c := range cs {
		c.Close()
	}
}

func (s *Server) addConn(conn net.Conn) {
	s.mu.Lock()
	s.conns[conn.RemoteAddr()] = conn
	s.mu.Unlock()
}

func (s *Server) removeConn(conn net.Conn) {
	s.mu.Lock()
	delete(s.conns, conn.RemoteAddr())
	s.mu.Unlock()
}
