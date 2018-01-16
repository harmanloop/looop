package protocol

import (
	"encoding/binary"
)

// Version is the version of protocol wire format
const Version byte = 0x01

var (
	// RawHeader - raw protocol header
	RawHeader = []byte{'l', 'o', 'p', Version}

	// Header - protocol header
	Header = binary.LittleEndian.Uint32(RawHeader)
)
