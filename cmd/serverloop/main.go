package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/harmanloop/looop"
)

func serve(s *looop.Server, ln net.Listener) <-chan error {
	ch := make(chan error)
	go func() {
		err := s.Serve(ln)
		ch <- err
	}()
	return ch
}

// Handle signals
func sigHandle() <-chan os.Signal {
	sigCaught := make(chan os.Signal)
	signal.Notify(sigCaught, syscall.SIGINT, syscall.SIGTERM)
	return sigCaught
}

func main() {
	fmt.Println("Server started!")
	ln, err := net.Listen("tcp", ":3377")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	s := looop.New()
	sigCh := sigHandle()
	select {
	case err := <-serve(s, ln):
		fmt.Println("serve finished:", err)
	case sig := <-sigCh:
		fmt.Println("signal caught:", sig)
		s.Stop()
	}
}
