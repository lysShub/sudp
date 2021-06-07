/* 传输策略 */
package strategy

import (
	"time"
)

/*
	速度控制基于当前速度，如果当前速度达到预期速度我们则增加新速度的值；如果没有达到预期速度则新速度是预期速度和实际速度的两倍。
*/

// 速度控制策略
var (
	SpeedTime  time.Duration = time.Millisecond * 500 // 发送速度控制包的周期 非局域网
	ResendTime time.Duration = time.Millisecond * 500 // 重发数据包检测周期 非局域网

	// SpeedTime  time.Duration = time.Millisecond * 500 // 发送速度控制包的周期 局域网
	// ResendTime time.Duration = time.Millisecond * 200 // 重发数据包检测周期 局域网

	delaylen  int   = 1                     // 速度记录器speeds的长度(>=1), 延时检测
	speeds    []int = make([]int, delaylen) // 设定速度记录器
	deviation int   = 5                     // 误差范围, 默认93.75% = (100-100/16)/100, 表示当前速度大于预定速度的93.75%即判定达到预期
	growRate  int   = 100                   // 增长率, 默认2^n增长，按照指数(l+growRate/100)^n倍增长
	nowSpeeds []int
)

// NewSpeed 更新速度
func NewSpeed(nowSpeed int) int {
	// 20 15.5
	// 30 19.8
	// 50 20.5
	// 100 18.7
	return 1048576 * 100

	if len(nowSpeeds) < 20 {
		nowSpeeds = append(nowSpeeds, nowSpeed)
	} else if len(nowSpeeds) == 20 {
		nowSpeeds = append(nowSpeeds, nowSpeed)
	}

	var ns int
	if nowSpeed == 0 {
		nowSpeed = 5120
	}

	if nowSpeed >= speeds[0] || speeds[0]/(speeds[0]-nowSpeed) > deviation { // 达到预期
		// 检测是否累加
		ns = nowSpeed + nowSpeed*growRate/100

	} else { // 未达预期
		if speeds[0]-nowSpeed > 1048576 {

			ns = nowSpeed + (speeds[0]-nowSpeed)>>1
		} else {
			ns = nowSpeed + 1024
		}
	}

	if len(speeds) < delaylen {
		speeds = append(speeds, ns)
	} else {
		speeds = append(speeds[1:], ns)
	}

	return ns
}

func max(d []int) int {
	var m int = 0
	for _, v := range d {
		if v > m {
			m = v
		}
	}
	return m
}

// CustomStratege 更新为自定义策略
func CustomStratege() error {
	return nil
}
