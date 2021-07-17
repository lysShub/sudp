package ioer

import (
	"io"
	"net"
)

// UDP的connected连接使得一个端口只能进行一个传输任务
// 通过raddr实现端口复用；使用一个端口即可同时进行多条传输任务，异或同时进行接收与发送任务
// 使用均listenUDP实现

// Conn 接收文件
type Conn struct {
	lconn *net.UDPConn // lister
	raddr *net.UDPAddr

	read *io.PipeReader
}

// Conn 读取数据
func (c *Conn) Read(b []byte) (int, error) {

	return c.read.Read(b)

}

// Write 发送数据
func (c *Conn) Write(b []byte) (int, error) {
	return c.lconn.WriteToUDP(b, c.raddr)
}

// Close 关键
func (c *Conn) Close() error {
	return c.Close()
}
