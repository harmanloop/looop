package looop

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/harmanloop/looop/protocol"
)

// A CodecBuilder creates a concrete Codec instance
type CodecBuilder interface {
	NewCodec(rw io.ReadWriter) Codec
}

// An Codec is the interface that wraps the Encode and Decode method
type Codec interface {
	Encoder
	Decoder
}

// An Encoder writes values to an output stream.
type Encoder interface {
	Encode(v interface{}) error
}

// A Decoder reades values from an input stream.
type Decoder interface {
	Decode(v interface{}) error
}

type Service func(net.Conn)

// Server is a generic server to serve IPC requests over
// various transport layers
type Server struct {
	ls         net.Listener
	serverWg   sync.WaitGroup
	clientWg   sync.WaitGroup
	quit       chan struct{}
	quitOnce   sync.Once
	serviceMap map[uint32]Service

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
	codec            CodecBuilder
}

var defaultOptions = options{
	h:                defaultHandshake,
	handshakeTimeout: time.Second * 3,
	codec:            defaultCodecBuilder{},
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

// WithCodec returns an Option which sets a codec for message
// marshalling and marshalling
func WithCodec(c CodecBuilder) Option {
	return func(o *options) {
		o.codec = c
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

func (s *Server) handleConn(conn net.Conn, c CodecBuilder) {
	var msg MessageKind
	codec := c.NewCodec(conn)

	for {
		err := codec.Decode(&msg)
		if err != nil {
			log.Printf("Decode error: %v", err)
			conn.Close()
			break
		}
		fn, ok := s.serviceMap[msg.Kind]
		if !ok {
			log.Printf("Unknown service id: %d", msg.Kind)
			continue
		}
		fn(conn)
	}
}

// Serve accepts incoming connections on the listener ls
func (s *Server) Serve(ls net.Listener) error {
	defer s.serverWg.Done()
	s.serverWg.Add(1)
	s.mu.Lock()
	s.ls = ls
	s.mu.Unlock()

	for {
		conn, err := s.ls.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return nil
			default:
			}
			return err
		}

		s.clientWg.Add(1)
		go func() {
			defer s.clientWg.Done()
			ok := s.opts.h(conn, s.opts.handshakeTimeout)
			if !ok {
				return
			}
			if !s.addConn(conn) {
				return
			}
			s.handleConn(conn, s.opts.codec)
			s.removeConn(conn)
		}()
	}
}

// Stop stops the server, closes all connections
func (s *Server) Stop() {
	s.quitOnce.Do(func() {
		close(s.quit)
		s.mu.Lock()
		s.ls.Close()
		s.mu.Unlock()
	})
	log.Println("Waiting for listener to close...")
	s.serverWg.Wait()

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
	log.Println("Waiting for connections to close...")
	s.clientWg.Wait()
}

// Register - registers a new service
func (s *Server) Register(id uint32, fn Service) error {
	_, present := s.serviceMap[id]
	if present {
		return fmt.Errorf("looop: service already defined: %d", id)
	}
	s.serviceMap[id] = fn
	return nil
}

// New creates a new instance of Server
func New(opt ...Option) *Server {
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}
	return &Server{
		quit:       make(chan struct{}),
		conns:      make(map[net.Addr]net.Conn),
		serviceMap: make(map[uint32]Service),
		opts:       opts,
	}
}

func (s *Server) addConn(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conns == nil {
		conn.Close()
		return false
	}
	s.conns[conn.RemoteAddr()] = conn
	return true
}

func (s *Server) removeConn(conn net.Conn) {
	s.mu.Lock()
	if s.conns != nil {
		delete(s.conns, conn.RemoteAddr())
	}
	s.mu.Unlock()
}

type defaultCodec struct {
	Encoder
	Decoder
}

type defaultCodecBuilder struct{}

func (c defaultCodecBuilder) NewCodec(rw io.ReadWriter) Codec {
	enc := gob.NewEncoder(rw)
	dec := gob.NewDecoder(rw)
	return defaultCodec{enc, dec}
}
