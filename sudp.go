package sudp

import (
	"net"
	"time"
)

type SUDP struct {
	Encrypt bool
	MTU     int
	TimeOut time.Duration

	conn *net.UDPConn
	key  []byte
}

var Version uint8 = 0b00000001
var err error
