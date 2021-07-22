// get ioer.Conn

package ioer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"
)

var errClosed error = errors.New("connection closed")

var listeners map[int64]*Listener // laddr Id

func init() {
	listeners = make(map[int64]*Listener)
}

// run 运行：1.路由接收到的UDP数据包; 2.接收新请求, 生成新的Conn
func (l *Listener) run() {

	var (
		id    int64
		c     *Conn
		ok    bool
		n     int
		raddr *net.UDPAddr
		err   error
	)
	for !l.done {
		if n, raddr, err = l.lconn.ReadFromUDP(l.tmp); !l.done && err != nil {
			errlog(err)

		} else if n > 0 {
			id = ider(raddr)

			if c, ok = l.conns[id]; !ok {
				var ch chan []byte = make(chan []byte, 16)

				c = new(Conn)
				c.io, c.lconn = ch, l.lconn
				c.raddr = raddr
				c.listenerid = ider(l.laddr)

				select {
				case l.rConn <- c: // 返回新Conn
				default:
				}
			}

			select {
			case c.io <- l.tmp[:n]: // 写入数据
			default:
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

// errlog 输出信息至os.Stderr
func errlog(err ...error) bool {

	var haveErr bool = false
	for i, e := range err {
		if e != nil {
			haveErr = true
			_, fp, ln, _ := runtime.Caller(1)
			if len(err) == 1 {
				fmt.Fprintln(os.Stderr, fp+":"+strconv.Itoa(ln)+" :\n    "+e.Error())
			} else {
				fmt.Fprintln(os.Stderr, "["+strconv.Itoa(i+1)+"]. "+fp+":"+strconv.Itoa(ln)+" \n    "+e.Error())
			}
		}
	}
	return haveErr
}
