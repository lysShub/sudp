package router_test

// 选择为数据路由的方式：存Map或存Slice（采用二分法查找）
import "testing"

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

var length int = 80

var k []int = make([]int, length)

var m map[int]int = make(map[int]int)

func init() {
	for i := 0; i < length; i++ {
		k[i], m[i] = i, i
	}
}

// getSlice
// 	@ T: 目标值
// 	返回对应的序号，没用找到返回-1
func getSlice1(T int) int {

	var x, y int = 0, len(k) - 1
	for {
		if y-x > 1 && k[(x+y)&0b1+(x+y)>>1] < T {
			x = (x+y)&0b1 + (x+y)/2

		} else if y-x > 1 && T < k[(x+y)>>1] {
			y = (x + y) / 2

		} else {

			if k[(x+y)&0b1+(x+y)>>1] == T {
				return (x+y)&0b1 + (x+y)>>1
			} else if k[(x+y)>>1] == T {
				return (x + y) >> 1
			} else {
				return -1
			}
		}
	}
}

func getSlice(T int) int {

	var x, y int = 0, len(k) - 1
	var s int
	for {
		s = x + y

		if y-x > 1 && k[s&0b1+s>>1] < T {
			x = s&0b1 + s/2

		} else if y-x > 1 && T < k[s>>1] {
			y = (x + y) / 2

		} else {

			if k[s&0b1+s>>1] == T {
				return s&0b1 + s>>1
			} else if k[s>>1] == T {
				return s >> 1
			} else {
				return -1
			}
		}
	}

}

func getMap(i int) int {
	return m[i]
}
