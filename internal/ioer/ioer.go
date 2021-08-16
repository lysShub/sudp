package ioer

import (
	"errors"
	"net"
	"sync"
)

// net.DialUDP会占用一个端口, 导致不能进行端口复用
// ioer的Dial和Accept实际均是使用net.ListenUDP实现
// 因此可以进行端口复用, 只要四元组中有一元不一样都可以进行连接

// Conn 表示一个链接
type Conn struct {
	lconn *net.UDPConn // unconnected net.UCPConn
	raddr *net.UDPAddr
	laddr *net.UDPAddr

	io         chan []byte // 读取数据管道
	listenerid int64       // 所属的listener
	done       bool        // Conn关闭flag
}

func Dial(laddr, raddr *net.UDPAddr) (*Conn, error) {

	var l *Listener
	var ok bool
	var err error

	if l, ok = listenersList[ider(laddr)]; ok {

		if c, ok := l.connList[ider(raddr)]; ok {
			return c, nil
		}
	} else if l, err = Listen(laddr); err != nil {
		return nil, err
	}

	var c = new(Conn)
	var ch chan []byte = make(chan []byte, 16)

	c.io = ch
	c.lconn = l.lconn
	c.raddr = raddr
	c.laddr = laddr
	c.listenerid = ider(laddr)

	l.Lock()
	l.connList[ider(raddr)] = c
	l.Unlock()

	return c, nil
}

// Read 读取数据; 确保b的长度足够大(65536), 否则会丢失部分数据
func (c *Conn) Read(b []byte) (int, error) {
	if c.done {
		return 0, errClosed
	} else {
		return copy(b, <-c.io), nil
	}
}

// Write 发送数据
func (c *Conn) Write(b []byte) (int, error) {
	if c.done {
		return 0, errClosed
	} else {
		return c.lconn.WriteToUDP(b, c.raddr)
	}
}

// Close 关闭
func (c *Conn) Close() error {

	if l, ok := listenersList[c.listenerid]; ok {
		l.Lock()
		delete(l.connList, ider(c.raddr))
		l.Unlock()
		c.done = true
		return nil
	} else {
		return nil
	}
}

var listenersList map[int64]*Listener // key为用laddr计算的Id

func init() {
	listenersList = make(map[int64]*Listener)
}

type Listener struct {
	connList     map[int64]*Conn // raddr Id
	sync.RWMutex                 // 锁, 凡是conns写的地方需要上锁

	lconn *net.UDPConn // net.ListenUDP
	laddr *net.UDPAddr // lconn的地址
	rConn chan *Conn   // 通信新生成的Conn
	done  bool         // 已关闭
}

// Listen 监听本地地址, 不会阻塞
func Listen(laddr *net.UDPAddr) (*Listener, error) {

	if laddr == nil || laddr.Port == 0 {
		return nil, errors.New("invalid laddr")
	} else if laddr.IP == nil {
		if lip, err := getLanIP(); err != nil {
			return nil, err
		} else {
			laddr.IP = lip
		}
	}

	if l, ok := listenersList[ider(laddr)]; ok {
		return l, nil
	}

	if conn, err := net.ListenUDP("udp4", laddr); err != nil {
		return nil, err
	} else {
		var l = new(Listener)
		var rConn chan *Conn = make(chan *Conn, 8)

		l.Lock()
		l.connList = make(map[int64]*Conn)
		l.Unlock()

		l.lconn = conn
		l.rConn = rConn
		listenersList[ider(laddr)] = l

		go l.run()

		return l, nil
	}
}

// Accept 接收请求, 会阻塞等待新的请求
func (l *Listener) Accept() *Conn {
	return <-l.rConn
}

// Close 会关闭所有链接
func (l *Listener) Close() error {
	l.done = true
	return l.lconn.Close()
}
