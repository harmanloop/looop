package protocol

import (
	"encoding/binary"
)

// Version is the version of protocol wire format
const Version byte = 0x01

var (
	// HeaderRaw - raw protocol header
	HeaderRaw = []byte{'l', 'o', 'p', Version}

	// Header - protocol header
	Header = binary.LittleEndian.Uint32(HeaderRaw)
)
