package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
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
		b := make([]byte, len(s))
		copy(b, s)
		srv.out <- b
		fmt.Println("read from srv:", string(<-srv.in))
	}
}
