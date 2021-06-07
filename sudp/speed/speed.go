package speed

import (
	"time"
)

/* 速度控制与重发周期 */

// 请使用New函数初始化
type Speed struct {
	SpeedPeriod  time.Duration          // 速度更新周期
	ResendPeriod time.Duration          // 重发检测周期
	NewSpeed     func(nowSpeed int) int // 根据当前速度返回设定速度

	nowSpeeds []int
	growRate  int
}

func New(f func(s *Speed) *Speed) *Speed {
	var s = new(Speed)
	s.NewSpeed = func(i int) int { return 0 }
	s = f(s)
	s.nowSpeeds = make([]int, 20)
	if s.SpeedPeriod == 0 {
		s.SpeedPeriod = time.Millisecond * 100
	}
	if s.ResendPeriod == 0 {
		s.ResendPeriod = time.Millisecond * 200
	}
	if s.NewSpeed(0) == 0 {
		s.NewSpeed = s.defaultNewSpeed
	}
	s.growRate = 50
	time.AfterFunc(time.Second*2, func() {
		// fmt.Println("快增长完成")
		s.growRate = 10
	})
	return s
}

// defaultNewSpeed 默认速度策略
func (s *Speed) defaultNewSpeed(nowSpeed int) int {
	if len(s.nowSpeeds) < cap(s.nowSpeeds) {
		s.nowSpeeds = append(s.nowSpeeds, nowSpeed)
	} else {
		s.nowSpeeds = append(s.nowSpeeds[1:], nowSpeed)
	}

	// return 1048576

	var ns int
	if nowSpeed <= 5120 { // 最小速度5KB
		ns = 5120
	} else {

		ns = nowSpeed + nowSpeed*s.growRate/100

		// ns = nowSpeed + s.averSpeed()*s.growRate/100 //s.averSpeed()

	}

	return ns
}

func (s *Speed) averSpeed() int {
	var avag int = 0
	for _, v := range s.nowSpeeds {
		avag += v
	}
	avag = avag / len(s.nowSpeeds)
	var avag2, t int
	for _, v := range s.nowSpeeds {
		if v >= avag {
			avag2 += v
			t++
		}
	}
	return avag2 / t
}
