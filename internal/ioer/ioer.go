package ioer

import (
	"errors"
	"net"
	"sync"
)

// UDP的connected连接使得一个端口只能进行一个传输任务
// 通过raddr实现端口复用；使用一个端口即可同时进行多条传输任务，异或同时进行接收与发送任务
// 使用均listenUDP实现

type Listener struct {
	conns        map[int64]*Conn // raddr Id
	sync.RWMutex                 // 锁, 凡是conns写的地方需要上锁

	lconn *net.UDPConn // net.ListenUDP
	laddr *net.UDPAddr // lconn的地址
	rConn chan *Conn   // 通信新生成的Conn
	tmp   []byte       // 临时存储
	done  bool         // 已关闭
}

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

	if l, ok := listeners[ider(laddr)]; ok {
		return l, nil
	}

	if conn, err := net.ListenUDP("udp4", laddr); err != nil {
		return nil, err
	} else {
		var l = new(Listener)
		var rConn chan *Conn = make(chan *Conn, 8)

		l.Lock()
		l.conns = make(map[int64]*Conn)
		l.Unlock()

		l.lconn = conn
		l.rConn = rConn
		l.tmp = make([]byte, 65536)
		listeners[ider(laddr)] = l

		go l.run()

		return l, nil
	}
}

// Accept 接收请求, 没用请求过来时可能会阻塞
func (l *Listener) Accept() *Conn {
	return <-l.rConn
}

// Close 会关闭所有链接
func (l *Listener) Close() error {
	l.done = true
	return l.lconn.Close()
}

// Conn 收发数据
type Conn struct {
	lconn *net.UDPConn // lister
	raddr *net.UDPAddr

	io         chan []byte
	listenerid int64 // Conn对应的Listener
	done       bool  // Conn关闭字段
}

func Dial(laddr, raddr *net.UDPAddr) (*Conn, error) {

	var l *Listener
	var ok bool
	var err error

	if l, ok = listeners[ider(laddr)]; ok {

		if c, ok := l.conns[ider(raddr)]; ok {
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
	c.listenerid = ider(laddr)

	l.Lock()
	l.conns[ider(raddr)] = c
	l.Unlock()

	return c, nil
}

// Conn 读取数据; 确保b的长度足够大(65536), 否则会丢失数据
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

	if l, ok := listeners[c.listenerid]; ok {
		l.Lock()
		delete(l.conns, ider(c.raddr))
		l.Unlock()
		c.done = true
		return nil
	} else {
		return nil
	}
}
