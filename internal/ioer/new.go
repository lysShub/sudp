// get ioer.Conn

package ioer

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

//  int64

// var readRouter map[int64]io.PipeWriter // ListenUDP
var readRouter map[int64]chan []byte // ListenUDP
var listeners map[int64]*Listener

func init() {
	// readRouter = make(map[int64]io.PipeWriter)
	readRouter = make(map[int64]chan []byte)
	listeners = make(map[int64]*Listener)
}

type Listener struct {
	lconn  *net.UDPConn
	tmp    []byte
	flagDo bool
}

//  &net.TCPAddr{IP: nil, Port: 19986}
func Listen(laddr *net.UDPAddr) (*Listener, error) {
	fmt.Println("Listen启动", laddr)

	if laddr == nil || laddr.Port == 0 {
		return nil, errors.New("invalid laddr")
	} else if laddr.IP == nil {
		if lip, err := getLanIP(); err != nil {
			return nil, err
		} else {
			laddr.IP = lip
		}
	}

	if l, ok := listeners[ider(laddr)]; ok {
		return l, nil
	}

	if conn, err := net.ListenUDP("udp4", laddr); err != nil {
		return nil, err
	} else {
		var l = new(Listener)
		l.lconn = conn
		l.tmp = make([]byte, 65535)
		listeners[ider(laddr)] = l
		return l, nil
	}
}

func (l *Listener) Accept(rCh chan *Conn) error {
	if rCh == nil {
		return errors.New("")
	}

	if l.flagDo { // 接管 重启

		l.flagDo = false
		laddr, err := net.ResolveUDPAddr("udp4", l.lconn.LocalAddr().String())
		if err != nil {
			return err
		}
		l.lconn.Close()

		l.lconn, err = net.ListenUDP("udp4", laddr)
		if err != nil {
			return err
		}
	}

	l.do(rCh)

	return nil

}

var flag sync.Once

func Dial(laddr, raddr *net.UDPAddr) (*Conn, error) {

	l, err := Listen(laddr)
	if err != nil {
		return nil, err
	}

	flag.Do(func() {
		go func() {
			l.do(nil)
		}()
	})

	raddr.IP = raddr.IP.To4()

	// var re *io.PipeReader
	// var wr *io.PipeWriter
	// re, wr = io.Pipe()

	var c = new(Conn)
	var ch chan []byte = make(chan []byte, 16)

	readRouter[ider(raddr)] = ch

	c.read = ch
	c.lconn = l.lconn
	c.raddr = raddr

	return c, nil
}

// do 用于Dail的read
func (l *Listener) do(rCh chan *Conn) {
	l.flagDo = true

	var (
		id    int64
		r     chan []byte
		ok    bool
		n     int
		raddr *net.UDPAddr
		err   error
	)
	for l.flagDo {
		if n, raddr, err = l.lconn.ReadFromUDP(l.tmp); err != nil {
			// fmt.Fprint(os.Stderr, err)
			panic(err)
		} else if n > 0 {
			id = ider(raddr)
			if r, ok = readRouter[id]; ok {
				// r.Write(l.tmp[:n]) // 写入可能会阻塞
				select {
				case r <- l.tmp[:n]:
				}

			} else if rCh != nil {

				// var re *io.PipeReader
				// var wr *io.PipeWriter
				// re, wr = io.Pipe()

				var ch chan []byte = make(chan []byte, 16)
				readRouter[id] = ch

				var c = new(Conn)
				c.read = ch
				c.lconn = l.lconn
				c.raddr = raddr
				rCh <- c
			}
		}
	}
}

func ider(addr *net.UDPAddr) int64 {
	if addr == nil {
		return 0
	} else {
		addr.IP = addr.IP.To16()
		if addr.IP == nil || len(addr.IP) < 16 {
			return int64(addr.Port)
		} else {
			return int64(addr.IP[12])<<+int64(addr.IP[13])<<32 + int64(addr.IP[14])<<24 + int64(addr.IP[15])<<16 + int64(addr.Port)
		}
	}
}

func getLanIP() (net.IP, error) {
	conn, err := net.DialTimeout("ip4:1", "8.8.8.8", time.Second*2)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(conn.LocalAddr().String()), nil
}
