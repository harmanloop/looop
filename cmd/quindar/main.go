package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/harmanloop/looop/nodeconn"
)

var state struct {
	sync.Mutex
	stopped bool
}

var quit = make(chan bool, 1)

func handleConnError(s *nodeconn.NodeConn) {
	fmt.Println("local err handler")
	state.Lock()
	if state.stopped == false {
		state.stopped = true
		s.Stop()
	}
	state.Unlock()
	quit <- true
}

func readStdin(srv *nodeconn.NodeConn, stdin *bufio.Scanner) {
	for stdin.Scan() {
		s := stdin.Bytes()
		if len(s) == 0 {
			continue
		}
		b := make([]byte, len(s))
		copy(b, s)
		srv.RawWrite(b)
		b, err := srv.RawRead()
		fmt.Printf("read from srv: %v, %v\n", b, err)
	}
	quit <- true
}

func main() {
	var wg sync.WaitGroup

	conn, err := net.Dial("tcp", ":3377")
	if err != nil {
		fmt.Println(err)
		return
	}
	srv := nodeconn.NewWithErrHandler(conn, &wg, handleConnError)
	wg.Add(2)
	go srv.PollRead()
	go srv.PollWrite()
	go readStdin(srv, bufio.NewScanner(os.Stdin))
	<-quit
	state.Lock()
	if state.stopped == false {
		state.stopped = true
		srv.Stop()
	}
	state.Unlock()
	wg.Wait()
}
