package ioer

import (
	"bytes"
	"io"
	"net"
)

// 端口聚合

type Conns struct {
	conns []*Conn
}

// Dial 增加宏端口中连接，并返回这个连接
func (cs *Conns) Dial(laddr, raddr *net.UDPAddr) (*Conn, error) {
	return Dial(laddr, raddr)
}

// Add 增加宏端口中的连接
func (cs *Conns) Add(c *Conn) {
	var laddr, raddr *net.UDPAddr = c.raddr, c.laddr
	for i := 0; i < len(cs.conns); i++ {
		if udpAddrEqual(laddr, cs.conns[i].laddr) && udpAddrEqual(raddr, cs.conns[i].raddr) {
			return
		}
	}
	cs.conns = append(cs.conns, c)
}

// Delete 删除并关闭连接
func (cs *Conn) Delete(c *Conn) {

}

// Read 读取
func (cs *Conns) Read() (int, error) {
	io.MultiReader()
	return 0, nil
}

// Write 写入
func (cs *Conns) Write() (int, error) {

	return 0, nil
}

func (cs *Conns) WriteTo() (int, error) {

	return 0, nil
}

func udpAddrEqual(r, l *net.UDPAddr) bool {
	if r.Port == l.Port && bytes.Equal(r.IP, l.IP) {
		return true
	}
	return false
}
