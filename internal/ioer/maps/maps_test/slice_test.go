package maps_test

// 比较map和数组(二分法)的读取速度

import "testing"

var length int = 80 // 数据量

// goos: windows
// goarch: amd64
// pkg: github.com/lysShub/ioer/maps/maps_test
// cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
// BenchmarkSlice-8   	24526435	        53.37 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMap-8     	24715513	        41.96 ns/op	       0 B/op	       0 allocs/op
// PASS
// ok  	github.com/lysShub/ioer/maps/maps_test	3.223s

func BenchmarkSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var j = i % (length - 1)
		if j != getSlice(j) {
			panic(j)
		}
	}
}

func BenchmarkMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var j = i % (length - 1)
		if j != getMap(j) {
			panic(j)
		}
	}
}

var L []int = make([]int, length)

var M map[int]int = make(map[int]int)

func init() {
	for i := 0; i < length; i++ {
		L[i], M[i] = i, i
	}
}

func getSlice(T int) int {

	var x, y int = 0, len(L) - 1
	var s int
	for {
		s = x + y

		if y-x > 1 && L[s&0b1+s>>1] < T {
			x = s&0b1 + s/2

		} else if y-x > 1 && T < L[s>>1] {
			y = (x + y) / 2

		} else {

			if L[s&0b1+s>>1] == T {
				return s&0b1 + s>>1
			} else if L[s>>1] == T {
				return s >> 1
			} else {
				return -1
			}
		}
	}

}

func getMap(i int) int {
	return M[i]
}
