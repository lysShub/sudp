package maps_test

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"os"
	"testing"

	"github.com/lysShub/ioer/maps"
)

// 测试maps和map的速度
// IP数据来源 https://github.com/17mon/china_ip_list

var addrsLenght int = 250 // map(s)中所有的数据量, 测试函数每次循环读写数据

// addrsLenght大小： 250
// goos: windows
// goarch: amd64
// pkg: github.com/lysShub/ioer/maps/maps_test
// cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
// BenchmarkMWrite-8    	16138857	        75.67 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMSWrite-8   	 7686552	       160.1 ns/op	      85 B/op	       0 allocs/op
// BenchmarkMRead-8     	20573872	        57.79 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMSRead-8    	21988532	        54.32 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	github.com/lysShub/ioer/maps/maps_test	6.037s

// addrsLenght大小： 6014
// goos: windows
// goarch: amd64
// pkg: github.com/lysShub/ioer/maps/maps_test
// cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
// BenchmarkMWrite-8    	11081028	       108.1 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMSWrite-8   	 6102097	       181.0 ns/op	      46 B/op	       0 allocs/op
// BenchmarkMRead-8     	12080844	       107.6 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMSRead-8    	 9198940	       134.9 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	github.com/lysShub/ioer/maps/maps_test	6.214s

func BenchmarkMWrite(b *testing.B) {
	for i := 0; i < b.N; i++ {

		m[ider(addrs[i%addrsLenght])] = i
	}
}

func BenchmarkMSWrite(b *testing.B) {
	for i := 0; i < b.N; i++ {

		ms.Add(addrs[i%addrsLenght], nil)
	}
}

func BenchmarkMRead(b *testing.B) {
	for i := 0; i < b.N; i++ {

		if _, ok := m[ider(addrs[i%addrsLenght])]; ok && false {
			fmt.Println()
		}
	}
}

func BenchmarkMSRead(b *testing.B) {
	for i := 0; i < b.N; i++ {

		if _, ok := ms.Read(addrs[i%addrsLenght]); ok && false {
			fmt.Println()
		}
	}
}

// ---------------------------------------------------- //

var m map[int64]int = make(map[int64]int)
var ms maps.Maps

var addrs []*net.UDPAddr = make([]*net.UDPAddr, 0, addrsLenght)

func init() {

	var sh *bufio.Scanner
	if fh, err := os.Open(`./ips.txt`); err != nil {
		panic(err)
	} else {
		sh = bufio.NewScanner(fh)
	}

	var randPort = func() string {
		r, err := rand.Int(rand.Reader, big.NewInt(65535))
		if err != nil {
			panic(err)
		}
		return r.String()
	}

	for i := 0; i < addrsLenght; i++ {
		if !sh.Scan() {
			addrsLenght = i - 1
		}
		if addr, err := net.ResolveUDPAddr("udp", sh.Text()+":"+randPort()); err != nil {
			continue
		} else {
			addrs = append(addrs, addr)
		}
	}

	fmt.Println("addrsLenght大小：", addrsLenght)
}

func ider(addr *net.UDPAddr) int64 {
	if addr == nil {
		panic("addr is nil")
	} else {
		addr.IP = addr.IP.To16()
		if addr.IP == nil || len(addr.IP) < 16 {
			return int64(addr.Port)
		} else {
			return int64(addr.IP[12])<<+int64(addr.IP[13])<<32 + int64(addr.IP[14])<<24 + int64(addr.IP[15])<<16 + int64(addr.Port)
		}
	}
}
