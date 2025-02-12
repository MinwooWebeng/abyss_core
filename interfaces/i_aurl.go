package interfaces

import (
	"net"
)

type AURL interface {
	ToString() string
	Scheme() string
	Hash() string
	Addresses() []*net.UDPAddr
	Path() string // does not include starting "/".
}
