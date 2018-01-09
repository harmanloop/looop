package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/harmanloop/looop/nodeconn"
)

func main() {
	var wg sync.WaitGroup

	conn, err := net.Dial("tcp", ":3377")
	if err != nil {
		fmt.Println(err)
		return
	}
	srv := nodeconn.New(conn, &wg)
	wg.Add(2)
	go srv.PollRead()
	go srv.PollWrite()

	stdin := bufio.NewScanner(os.Stdin)
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
	srv.Close()
	wg.Wait()
}
