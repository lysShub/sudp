package ioer

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
)

// var pool map[int64]int64
var pool map[[6]byte]io.PipeWriter // ListenUDP
var listeners map[[6]byte]*Listener

func init() {
	pool = make(map[[6]byte]io.PipeWriter)
	listeners = make(map[[6]byte]*Listener)
}

type Listener struct {
	lconn  *net.UDPConn
	buffer []byte
	flagDo bool
}

//  &net.TCPAddr{IP: nil, Port: 19986}
func Listen(laddr *net.UDPAddr) (*Listener, error) {
	fmt.Println("Listen启动", laddr)

	if l, ok := listeners[ider(laddr)]; ok {
		fmt.Println("Listen已经存在")
		return l, nil
	}

	if conn, err := net.ListenUDP("udp4", laddr); err != nil {
		return nil, err
	} else {
		var l = new(Listener)
		l.lconn = conn
		l.buffer = make([]byte, 65535)
		return l, nil
	}
}

func (l *Listener) Accept(rCh chan *Conn) error {
	if rCh == nil {
		return errors.New("")
	}

	if l.flagDo { // 接管 重启
		fmt.Println("接管")

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

	var (
		id    [6]byte
		r     io.PipeWriter
		ok    bool
		n     int
		raddr *net.UDPAddr
		err   error
	)

	for {
		if n, raddr, err = l.lconn.ReadFromUDP(l.buffer); err != nil {
			panic(err)
		} else if n > 0 {
			fmt.Println(raddr, "收到")

			id = ider(raddr)
			if r, ok = pool[id]; ok {
				fmt.Println("存在")
				r.Write(l.buffer[:n])

			} else {
				fmt.Println("不存在")

				var re *io.PipeReader
				var wr *io.PipeWriter
				re, wr = io.Pipe()

				pool[id] = *wr

				var c = new(Conn)
				c.read = re
				c.lconn = l.lconn
				c.raddr = raddr
				rCh <- c
			}
			fmt.Println("完成")
		}
	}
}

var flag sync.Once

func Dial(laddr, raddr *net.UDPAddr) (*Conn, error) {

	l, err := Listen(laddr)
	if err != nil {
		return nil, err
	}

	flag.Do(func() {
		go func() {
			l.do()
		}()
	})

	raddr.IP = raddr.IP.To4()

	var c = new(Conn)
	var re *io.PipeReader
	var wr *io.PipeWriter
	re, wr = io.Pipe()

	pool[ider(raddr)] = *wr

	c.read = re
	c.lconn = l.lconn
	c.raddr = raddr

	return c, nil
}

// do 用于Dail的read
func (l *Listener) do() {
	l.flagDo = true

	var (
		id    [6]byte
		r     io.PipeWriter
		ok    bool
		n     int
		raddr *net.UDPAddr
		err   error
	)
	for l.flagDo {
		if n, raddr, err = l.lconn.ReadFromUDP(l.buffer); err != nil {
			// fmt.Fprint(os.Stderr, err)
			panic(err)
		} else if n > 0 {
			id = ider(raddr)
			if r, ok = pool[id]; ok {
				r.Write(l.buffer[:n])
			}
		}
	}
}

func ider(addr *net.UDPAddr) [6]byte {
	if addr == nil {
		return [6]byte{0, 0, 0, 0, 0, 0}
	} else {
		addr.IP = addr.IP.To16()

		if addr.IP == nil || len(addr.IP) < 16 {
			fmt.Println("不合法IP", addr.IP)
			return [6]byte{0, 0, 0, 0, byte(addr.Port >> 8), byte(addr.Port)}
		} else {
			return [6]byte{addr.IP[12], addr.IP[13], addr.IP[14], addr.IP[15], byte(addr.Port >> 8), byte(addr.Port)}
		}
	}
}
