/*
* 弃用, 与原生map相比并没用明显优势; 参考maps_test结果
 */

// 专为net.Addr设计的map, 也可以使用原生map[int64]*ioer.Conn
// 一个地址ip4:port总共48位, 可以使用int64存放

package maps

import (
	"net"
	"sync"
	"time"

	"github.com/lysShub/ioer"
)

var defaultIP net.IP

func init() {
	defaultIP = localIP()
}

type b struct {
	ipd int64
	c   *ioer.Conn
}

type Maps struct {
	h    [65535][]b
	lock sync.RWMutex
}

// Add 追加
func (a *Maps) Add(addr *net.UDPAddr, c *ioer.Conn) {
	if addr.IP == nil {
		addr.IP = defaultIP
	} else if len(addr.IP) < 16 {
		addr.IP = addr.IP.To16()
	}

	var ipd int64 = int64(addr.IP[12])<<+int64(addr.IP[13])<<32 + int64(addr.IP[14])<<24 + int64(addr.IP[15])<<16 + int64(addr.Port)

	// HASH
	// 8 8 8 8 16
	// 2   2 8  4
	// var k uint16 = uint16(((ipd>>40)&0x3)<<14 + ((ipd>>24)&0x3)<<12 + ((ipd>>16)&0xff)<<4 + ipd&0xf)
	var k uint16 = uint16((ipd>>26)&0xC000 + (ipd>>12)&0x3000 + (ipd>>12)&0xff0 + ipd&0xf)

	a.lock.Lock()
	if a.h[k] == nil {
		a.h[k] = make([]b, 0, 8)
	}
	a.h[k] = append(a.h[k], b{ipd: ipd, c: c})
	a.lock.Unlock()
}

// Read 读取
func (a *Maps) Read(addr *net.UDPAddr) (*ioer.Conn, bool) {
	if addr.IP == nil {
		addr.IP = defaultIP
	} else if len(addr.IP) < 16 {
		addr.IP = addr.IP.To16()
	}

	var ipd int64 = int64(addr.IP[12])<<+int64(addr.IP[13])<<32 + int64(addr.IP[14])<<24 + int64(addr.IP[15])<<16 + int64(addr.Port)
	var k uint16 = uint16((ipd>>26)&0xC000 + (ipd>>12)&0x3000 + (ipd>>12)&0xff0 + ipd&0xf)

	if a.h[k] == nil {
		return nil, false
	} else {
		for _, v := range a.h[k] {
			if v.ipd == ipd {
				return v.c, true
			}
		}
		return nil, false
	}
}

func (a *Maps) Delete(addr *net.UDPAddr) {
	if addr.IP == nil {
		addr.IP = defaultIP
	} else if len(addr.IP) < 16 {
		addr.IP = addr.IP.To16()
	}

	var ipd int64 = int64(addr.IP[12])<<+int64(addr.IP[13])<<32 + int64(addr.IP[14])<<24 + int64(addr.IP[15])<<16 + int64(addr.Port)
	var k uint16 = uint16((ipd>>26)&0xC000 + (ipd>>12)&0x3000 + (ipd>>12)&0xff0 + ipd&0xf)

	if a.h[k] == nil {
		return
	} else {
		for n, v := range a.h[k] {
			if v.ipd == ipd {
				a.lock.Lock()
				a.h[k] = append((a.h[k])[:n], (a.h[k])[n+1:]...)
				a.lock.Unlock()
				return
			}
		}
	}
}

func localIP() net.IP {
	conn, err := net.DialTimeout("ip4:1", "8.8.8.8", time.Second)
	if err != nil {
		return net.ParseIP("0.0.0.0").To16()
	}
	return net.ParseIP(conn.LocalAddr().String()).To16()
}
