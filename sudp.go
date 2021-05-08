package sudp

import (
	"net"
	"time"
)

type SUDP struct {
	Encrypt bool
	MTU     int

	conn    *net.UDPConn
	timeOut time.Duration
	key     []byte
}

var Version uint8 = 0b00000001
var err error
