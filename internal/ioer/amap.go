// 专为net.Addr设计的map

package ioer

// 只支持IPv4
// 由于暂时不考虑单连接多链路，所以Addr的端口是确定的
// hash方法： ip[3]<<8+ip[4]

type amap struct {
}

// 储存h
type h = [65536]int

var mapk []h = make([][65536]int, 0, 10)

// 储存v, mapv[0]不使用
//  会一直累积, 超过容量可能会发生垃圾扫描
var mapv []*Conn = make([]*Conn, 1, 256)
