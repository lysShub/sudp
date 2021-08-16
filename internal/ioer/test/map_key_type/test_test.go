package test_test

import (
	"net"
	"testing"
)

// 测试不同类型的map的key效率
// net.UDPAddr由于不能进行`=`操作, 所以不能作为map的key; 可以把net.UDPAddr映射为
// 数组或int64

// goos: windows
// goarch: amd64
// pkg: github.com/lysShub/ioer/test/map_key_type
// cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
// BenchmarkInt-8     	72434024	        14.44 ns/op	       0 B/op	       0 allocs/op
// BenchmarkArray-8   	18692054	        68.13 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	github.com/lysShub/ioer/test/map_key_type	2.504s

var addr *net.UDPAddr = &net.UDPAddr{IP: net.ParseIP("23.56.156.245").To16(), Port: 19986}

var intMap map[int64]struct{} = make(map[int64]struct{})
var arrMap map[[5]int]struct{} = make(map[[5]int]struct{})

func init() {
	var intaddr = int64(addr.IP[12])<<+int64(addr.IP[13])<<32 + int64(addr.IP[14])<<24 + int64(addr.IP[15])<<16 + int64(addr.Port)
	var arraddr = [5]int{int(addr.IP[12]), int(addr.IP[13]), int(addr.IP[14]), int(addr.IP[15]), addr.Port}

	intMap[intaddr] = struct{}{}
	arrMap[arraddr] = struct{}{}
}

func BenchmarkInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var k int64 = int64(addr.IP[12])<<+int64(addr.IP[13])<<32 + int64(addr.IP[14])<<24 + int64(addr.IP[15])<<16 + int64(addr.Port)
		if _, ok := intMap[k]; ok {

		}
	}
}

func BenchmarkArray(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var k [5]int = [5]int{int(addr.IP[12]), int(addr.IP[13]), int(addr.IP[14]), int(addr.IP[15]), addr.Port}
		if _, ok := arrMap[k]; ok {

		}
	}
}
