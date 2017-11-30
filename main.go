// Package main provides ...
package main

import (
	bin "encoding/binary"
	"fmt"
	"log"
	"net"
)

const protoVersion byte = 0x01

var (
	magic = bin.LittleEndian.Uint32([]byte{'l', 'o', 'p', protoVersion})
)

type pktHdr struct {
	magic   []byte
	version byte
	length  uint32
}

func main() {
	fmt.Println("looop is preparing to disrupt the industry!")

	addr, err := net.ResolveTCPAddr("tcp4", ":3377")
	if err != nil {
		log.Fatalf("Could not resolve: %v\n", err)
	}

	listener, err := net.ListenTCP("tcp4", addr)
	_ = listener
	if err != nil {
		log.Fatalf("Could not listen: %v\n", err)
	}

	bb := make([]uint8, 4)
	ret := bin.PutUvarint(bb, uint64(0xab))
	fmt.Printf("ret: %d, %#v\n", ret, bb)
	for i := range bb {
		fmt.Printf("%x\n", bb[i])
	}
	bc := make([]byte, 2)
	bin.BigEndian.PutUint16(bc, 0xabcd)
	fmt.Printf("bigend: %#v\n", bc)

	le := make([]byte, 2)
	bin.LittleEndian.PutUint16(le, 0xabcd)
	fmt.Printf("little: %#v\n", le)
	fmt.Printf("%#v\n", magic)
	// conn, err := listener.Accept()
	// conn.Write([]byte("What the fuck!\n"))
}
