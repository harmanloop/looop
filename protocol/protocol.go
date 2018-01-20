package protocol

import (
	"encoding/binary"
)

const (
	// Version is the version of protocol wire format
	Version byte = 0x01

	// HdrLen is the length of header in bytes
	HdrLen = 4
)

var (
	// RawHeader - raw protocol header
	RawHeader = []byte{'l', 'o', 'p', Version}

	// Header - protocol header
	Header = binary.LittleEndian.Uint32(RawHeader)
)

// ValidHdr - return true if the p represents a valid header, and false
// otherwise
func ValidHdr(p []byte) bool {
	u := binary.LittleEndian.Uint32(p)
	return u == Header
}
