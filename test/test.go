package main

import (
	"fmt"
	"net"
	"time"

	"github.com/lysShub/sudp/internal/ioer"
)

// var key = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}
var key []byte = nil

var blockSize int = 1204

func main() {

	var ip net.IP = net.ParseIP("a.b.c.d:p")

	//      a    b    c    d     p
	var m [][][][][]int = make([][][][][]int, 256)

	m[170][168][192][9][323] = 99
	fmt.Println(ip)
	time.Sleep(time.Hour)
	return
	// client 请求
	go Client()
	time.Sleep(time.Second)

	// 开启一个sever 19986
	l, err := ioer.Listen(&net.UDPAddr{IP: net.ParseIP("192.168.43.179"), Port: 19986})
	if err != nil {
		panic(err)
	}

	var rCh chan *ioer.Conn = make(chan *ioer.Conn)
	go l.Accept(rCh)

	for tconn := range rCh {
		var conn *ioer.Conn = tconn
		go func() {
			var da []byte = make([]byte, 1200)
			for {
				if n, err := conn.Read(da); err != nil {
					panic(err)
				} else {
					fmt.Println("sever收", string(da[:n]))
					conn.Write([]byte("大师傅撒发生"))
				}
			}
		}()
	}
	fmt.Println("退出")
}

func Client() {
	conn, err := ioer.Dial(&net.UDPAddr{IP: net.ParseIP("192.168.43.179"), Port: 19987}, &net.UDPAddr{IP: net.ParseIP("192.168.43.179"), Port: 19986})
	if err != nil {
		panic(err)
	}
	go func() { // 收
		var da []byte = make([]byte, 2000)
		for {
			if n, err := conn.Read(da); err != nil {
				panic(err)
			} else {
				fmt.Println("client收：", string(da[:n]))
			}
		}
	}()
	for {
		conn.Write([]byte("sadfsadfsa"))
		time.Sleep(time.Second)
	}
}
