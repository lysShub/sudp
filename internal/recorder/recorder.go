package recorder

import (
	"sync"
)

type Recorder struct {
	// 一个写入记录器

	cover   bool    //有覆盖写入
	rec     []int64 // 写入记录
	addChan chan [2]int64
	end     bool // 结束标志, 用于退出协程
	lock    sync.RWMutex
}

// NewRecorder 初始记录器
func (r *Recorder) NewRecorder() {
	r.rec = make([]int64, 0, 64)
	r.addChan = make(chan [2]int64, 16)
	r.end = false
	r.cover = true

	go func() {
		var adds [2]int64
		var c bool
		for !r.end {
			adds = <-r.addChan
			r.lock.Lock()
			r.rec, c = recorder(r.rec, adds[0], adds[1])
			r.lock.Unlock()
			if c && !r.cover {
				r.cover = true
			}

		}
	}()
	// time.Sleep(time.Millisecond * 1000)
}

// Add 增加
func (r *Recorder) Add(start, end int64) {
	r.addChan <- [2]int64{
		start, end,
	}
}

// End 结束记录器, 用于退出协程
func (r *Recorder) End() {
	r.end = true
}

// HasCover 当前是否有覆盖
func (r *Recorder) HasCover() bool {
	return r.cover
}

// Shche 进度, 0开头的块的结尾
func (r *Recorder) Shche() int64 {
	if len(r.rec) > 0 && r.rec[0] == 0 {
		return r.rec[1]
	}
	return 0
}

// Expose 暴露记录切片
func (r *Recorder) Expose() []int64 {
	return r.rec
}

// Sum 统计写入的总数据
func (r *Recorder) Sum() int64 {
	var s int64
	r.lock.Lock()
	for i := 0; i <= len(r.rec)-2; i = i + 2 {
		s = s + (r.rec[i+1] - r.rec[i])
	}
	r.lock.Unlock()
	return s
}

// Blocks 当前的块数
//  当写入完全后, rec应该只有1个块, 2个数据
func (r *Recorder) Blocks() int64 {
	var s int64
	r.lock.Lock()
	s = int64(len(r.rec) / 2)
	r.lock.Unlock()
	return s
}

// Owe 统计缺失文件, 最多返回100组数据
func (r *Recorder) Owe() [][2]int64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	var l int = len(r.rec)
	if l < 4 {
		return nil
	}

	var R [][2]int64

	for i := 2; i < l-1 && i <= 200; i = i + 2 {
		R = append(R, [2]int64{
			r.rec[i-1] + 1, r.rec[i] - 1,
		})
	}
	return R
}

// Owe 统计缺失文件总和, 返回所有缺失数据
func (r *Recorder) OweAll() [][][2]int64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	var l int = len(r.rec)
	if l < 4 {
		return nil
	}

	var R [][][2]int64
	var t [][2]int64

	for i := 2; i < l-1; i = i + 2 {
		t = append(t, [2]int64{
			r.rec[i-1] + 1, r.rec[i] - 1,
		})
		if i%200 == 0 { //
			var tmp [][2]int64 = make([][2]int64, len(t))
			copy(tmp, t)
			R = append(R, tmp)
			t = nil
		}
	}
	R = append(R, t)

	return R
}

// SumSpecify 统计从指定位置开始缺失数据总和所占万分比的数值
func (r *Recorder) OweSpecify(start int64, CountRange int) int {
	r.lock.Lock()
	defer r.lock.Unlock()
	var l int = len(r.rec)
	end := r.rec[l-1] - int64(CountRange) // 统计边界
	// end := r.rec[l-1] // 统计边界
	var O int64 // 缺失

	for i := 1; i <= l-2; i = i + 2 {
		if r.rec[i] >= start {
			for j := i; j < l-2; j++ {
				if r.rec[j] >= end {
					return int(O * 1e4 / (end - start))
				} else {
					O = O + r.rec[j+1] - r.rec[j] - 1
				}
			}
			return int(O * 1e4 / (end - start))
		}
	}
	return 0
}

func recorder(rec []int64, start, end int64) ([]int64, bool) {
	var l int = len(rec)
	if l == 0 {
		rec = append(rec, start, end)
		return rec, false
	}

	var tmp []int64 = make([]int64, 0, l)
	var ex bool = false
	if rec[l-1]+1 == start { //	//绝大多数情况
		tmp = rec
		tmp[l-1] = end
	} else { //覆盖所有情况

		var max func(x, y int64) int64 = func(x, y int64) int64 {
			if x > y {
				return x
			}
			return y
		}
		var min func(x, y int64) int64 = func(x, y int64) int64 {
			if x < y {
				return x
			}
			return y
		}
		var merged bool = false

		for i := 0; i < l; i = i + 2 {
			if rec[i]-1 > end {
				if !merged {
					tmp = append(tmp, start, end)
					merged = true
				}
				tmp = append(tmp, rec[i], rec[i+1])

			} else if rec[i+1]+1 < start {
				tmp = append(tmp, rec[i], rec[i+1])

			} else { //有重复区间
				start = min(start, rec[i])
				end = max(end, rec[i+1])
				ex = true
			}
		}
		if !merged {
			tmp = append(tmp, start, end)
		}
	}
	return tmp, ex
}

/* -------------速度------------- */
