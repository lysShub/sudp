package tasker

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sudp"
	"sudp/internal/crypter"
	"sudp/internal/file"
	"sudp/internal/packet"
	"sudp/internal/recorder"
	"time"

	"github.com/lysShub/e"
)

type Receiver struct {
	Speed        int // 实时速度，B/s
	DefaultSpeed int // 默认速度, B/s

	/* ----------------------------- */
	key          []byte        // 密钥
	conn         *net.UDPConn  // 发送方初始为unconnected UDPConn
	matchTimeOut time.Duration // 匹配超时, 一般设置较长时间
	replyTimeOut time.Duration // 回复超时时间, 较短时间
	mtu          int           // 链路MTU
	forwardBias  int64         // 最前的bias
	cspeed       int64         // 实时速度更新周期内接收到的数据包计数器

	speedStrategy
}

// speedStrategy 速度控制策略
type speedStrategy struct {
	// statPeriod time.Duration // 重发检测和速度控制的周期
	// countRange int           // 检测范围，与当前速度、重发周期有关，通常为n个传输周期内传输数据的大小

	/*
	* 速度变化条件：速度未达到期望速度或前f个速度同号。不满足变换条件时，保持原有速度。
	* 速度变化规律：没有发现发现实际带宽带宽时，速度以指数变化；发现实际带宽时，速度以固定量变化。
	* 达到期望速度：当前速度与上一个速度误差在6%以内
	* 发现实际带宽：当前f个数据相邻相互异号时
	 */

	// 速度记录器，记录之前的n个速度大小
	speedsRecorder []int

	// 反馈长度, 当前f速度符号相同(同增或同减)，那么速度就进行响应的变换。
	feedbackLength int
}

func (r *Receiver) rinit() {
	if r.DefaultSpeed <= 1 {
		r.DefaultSpeed = 131072 // 0.125MB/s
	}
	r.matchTimeOut = time.Minute * 15
	r.replyTimeOut = time.Second
	r.mtu = 1372

	r.feedbackLength = 3
	for i := 0; i < r.feedbackLength; i++ {
		r.speedsRecorder = append(r.speedsRecorder, r.DefaultSpeed)
	}

}

// rS3FFFFF0000 发送任务请求包
func (r *Receiver) rS3FFFFF0000(requestBody []byte, laddr, raddr *net.UDPAddr) error {
	if r.conn, err = net.DialUDP("udp", laddr, raddr); e.Errlog(err) {
		return err
	}

	var p []byte
	if p, _, _, err = packet.PackageDataPacket(requestBody, 0x3FFFFF0000, nil, false); err != nil {
		return err
	} else {
		for i := 0; i < reliable; i++ {
			if _, err = r.conn.Write(p); err != nil {
				return err
			}
		}
	}
	return nil
}

// rR3FFFFF8000 接收任务握手包
//  返回 确认握手包 私钥(不需加密是nil)
func (r *Receiver) rR3FFFFF8000() ([]byte, []byte, error) {
	var ch chan error = make(chan error, 1)
	var flag bool = true
	var p, da, priKey []byte

	go func() { // 接收握手包并发送确认握手包

		for flag {
			da = make([]byte, 1500)
			if err = r.conn.SetReadDeadline(time.Now().Add(time.Second)); e.Errlog(err) {
				ch <- err // 不再读取基于超时
				return
			}
			if n, err := r.conn.Read(da); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					ch <- errors.New("timeout")
				} else {
					ch <- err
				}
				return
			} else if dl, bias, _, err := packet.ParseDataPacket(da[:n], nil); err == nil {
				da = da[:dl]
				if bias == 0x3FFFFF8000 {
					if da[0] != sudp.Version { // 版本不相同
						p = []byte{10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 252, 0, 8, 106, 249, 147, 14}
						priKey = nil
						ch <- errors.New("incompatible protocol version")
						return
					} else {
						mtu := int(da[1])<<8 + int(da[2])
						if mtu < r.mtu {
							r.mtu = mtu
							fmt.Println("mtu", mtu)
						}
						if da[3] != 0 { // 加密

							var pubkey []byte
							if priKey, pubkey, err = crypter.RsaGenKey(); err != nil {
								ch <- err
								return
							} else {
								if p, _, _, err = packet.PackageDataPacket(append([]byte{0, uint8(r.mtu >> 8), uint8(r.mtu)}, pubkey...), 0x3FFFFF4000, nil, false); err != nil {
									ch <- err
									return
								}
							}

						} else {
							priKey = nil
							if p, _, _, err = packet.PackageDataPacket(append([]byte{0, uint8(r.mtu >> 8), uint8(r.mtu)}, make([]byte, 162)...), 0x3FFFFF4000, nil, false); err != nil {
								ch <- err
								return
							}
						}
						ch <- nil
						return
					}

				}
			} else {
				e.Errlog(err)
			}

		}

	}()

	select {
	case err = <-ch:
		if err == nil {
			return p, priKey, nil
		}
		return nil, nil, err
	case <-time.After(r.replyTimeOut * 2):
		flag = false
		return nil, priKey, errors.New("timeout")
	}
}

// rS3FFFFF4000 回复确认握手包
func (r *Receiver) rS3FFFFF4000(p []byte) error {
	for i := 0; i < reliable; i++ {
		if _, err = r.conn.Write(p); e.Errlog(err) {
			return err
		}
	}
	return nil
}

// rR3FFFFF2000 接收确认确认握手包
func (r *Receiver) rR3FFFFF2000(privateKey []byte) error {
	var ch chan error = make(chan error, 1)
	var flag bool = true
	var da []byte

	go func() { // 接收握手包
		for flag {
			da = make([]byte, 1500)
			if err = r.conn.SetReadDeadline(time.Now().Add(r.replyTimeOut)); e.Errlog(err) {
				ch <- err
				return
			}
			if n, err := r.conn.Read(da); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					ch <- errors.New("timeout")
				} else {
					ch <- err
				}
				return
			} else if dl, bias, _, err := packet.ParseDataPacket(da[:n], nil); err == nil {
				if bias == 0x3FFFFF2000 {
					if privateKey != nil {
						if rkey, err := crypter.RsaDecrypt(da[:dl], privateKey); e.Errlog(err) {
							ch <- err
							return
						} else {
							r.key = rkey
						}
					} else {
						r.key = nil
					}
					ch <- nil
					return

				}

			} else {
				e.Errlog(err)
			}

		}

	}()

	select {
	case err = <-ch:
		return err
	case <-time.After(r.replyTimeOut * 2):
		flag = false
		return errors.New("timeout")
	}
}

// rS3FFFFF1000 发送任务开始包
func (r *Receiver) rS3FFFFF1000() error {
	var da []byte
	if da, _, _, err = packet.PackageDataPacket(nil, 0x3FFFFF1000, nil, false); e.Errlog(err) {
		return err
	}
	for i := 0; i < 20; i++ { // 回复开始包
		if _, err = r.conn.Write(da); e.Errlog(err) {
			return err
		}
	}
	return nil
}

/* ------------------------------------------------- */

// rR3FFFFF0001OR3FFFFFFF00 接收文件信息包或任务结束包
//  返回文件名; 文件大小; 是否任务结束包
//  为默认值且无错误是为任务结束包
func (r *Receiver) rR3FFFFF0001OR3FFFFFFF00() (string, int64, bool, error) {
	var name string
	var fi int64
	var ch chan error = make(chan error, 1)
	var flag bool = true
	var end bool = false

	go func() {
		var da []byte
		var l int
		var bias, dl int64
		for flag {
			da = make([]byte, 1500)
			if err = r.conn.SetReadDeadline(time.Now().Add(r.replyTimeOut)); e.Errlog(err) {
				ch <- err
				return
			}
			if l, err = r.conn.Read(da); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					ch <- errors.New("timeout")
				} else {
					ch <- err
				}
				return
			}
			if dl, bias, _, err = packet.ParseDataPacket(da[:l], r.key); err == nil {
				if bias == 0x3FFFFF0001 { // 文件信息
					fmt.Println("接收到文件信息包")

					fi = int64(da[0])<<32 + int64(da[1])<<24 + int64(da[2])<<16 + int64(da[3])<<8 + int64(da[4])
					name = string(da[5:dl])
					fmt.Println("name", name)
					ch <- nil
					return
				} else if bias == 0x3FFFFFFF00 { // 任务结束包
					end = true
					ch <- nil
					return
				}

			}
		}
	}()

	select {
	case err = <-ch:
		if err == nil {
			return name, fi, end, nil
		}
		return "", 0, false, err
	case <-time.After(r.replyTimeOut * 2):
		flag = false
		return "", 0, false, errors.New("timeout")
	}
}

// rS3FFFFF0002 回复文件开始包
func (r *Receiver) rS3FFFFF0002() error {
	var da []byte
	if da, _, _, err = packet.PackageDataPacket(nil, 0x3FFFFF0002, r.key, false); e.Errlog(err) {
		return err
	}
	for i := 0; i < reliable; i++ { // 回复开始包
		if _, err = r.conn.Write(da); e.Errlog(err) {
			return err
		}
	}
	return nil
}

// rS3FFFFF00FF 发送文件结束包
func (r *Receiver) rS3FFFFF00FF() error {
	var da []byte
	if da, _, _, err = packet.PackageDataPacket(nil, 0x3FFFFF00FF, r.key, false); e.Errlog(err) {
		return err
	}
	for i := 0; i < reliable; i++ { // 回复开始包
		if _, err = r.conn.Write(da); e.Errlog(err) {
			return err
		}
	}
	return nil
}

// sS3FFFFF0004 回复文件重发包
func (r *Receiver) rS3FFFFF0004(ownRec [][2]int64) error {
	if len(ownRec) == 0 {
		return nil
	}
	fmt.Println(ownRec)

	var da []byte = make([]byte, 0)
	for _, v := range ownRec {
		da = append(da, uint8((v[0])>>32), uint8((v[0])>>24), uint8((v[0])>>16), uint8((v[0])>>8), uint8((v[0])), uint8((v[1])>>32), uint8((v[1])>>24), uint8((v[1])>>16), uint8((v[1])>>8), uint8((v[1])))
	}

	if da, _, _, err = packet.PackageDataPacket(da, 0x3FFFFF0004, r.key, false); e.Errlog(err) {
		return err
	}
	if _, err = r.conn.Write(da); e.Errlog(err) {
		return err
	}

	return nil
}

// sS3FFFFF0008 回复文件进度包
func (r *Receiver) rS3FFFFF0008(s int64) error {
	var da []byte
	if da, _, _, err = packet.PackageDataPacket([]byte{uint8(s >> 32), uint8(s >> 24), uint8(s >> 16), uint8(s >> 8), uint8(s)}, 0x3FFFFF0008, r.key, false); e.Errlog(err) {
		return err
	}
	if _, err = r.conn.Write(da); e.Errlog(err) {
		return err
	}
	return nil
}

// rS3FFFFF0010 速度控制包
//  是否为正数 速度变量
func (r *Receiver) rS3FFFFF0010(ns int) error {
	var da []byte = []byte{uint8(ns >> 24), uint8(ns >> 16), uint8(ns >> 8), uint8(ns)}
	if da, _, _, err = packet.PackageDataPacket(da, 0x3FFFFF0010, r.key, false); e.Errlog(err) {
		return err
	}
	if _, err = r.conn.Write(da); e.Errlog(err) {
		return err
	}
	return nil
}

// 接收文件数据数据包
func (r *Receiver) rRFileDataPacket(fh *os.File, fi int64) error {
	w := new(file.Wt) // 写入器
	w.Fh = fh

	rec := new(recorder.Recorder) // 记录器
	defer rec.End()
	rec.NewRecorder()

	var da []byte = make([]byte, 1500)
	var l int = 0
	var dl, bias int64
	var end, tend, flag = false, false, true // 接收到最后包, _ , 结束传输
	var ch chan error = make(chan error)

	go func() { // 速度
		for flag { // 周期
			time.Sleep(time.Second)

			// 速度控制
			n := r.newSpeed()
			fmt.Println("速度", n, r.Speed)

			if err = r.rS3FFFFF0010(n); e.Errlog(err) {
				fmt.Println(err)
			}
		}
	}()

	go func() { // 重发
		for flag {
			time.Sleep(time.Millisecond * 500)
			if re := rec.Owe(0); len(re) > 0 || end {
				if err = r.rS3FFFFF0004(re); e.Errlog(err) {
					ch <- err
					return
				}
				if rec.Blocks() == 1 {
					fmt.Println("文件传输完成")
					if rec.HasCover() {
						e.Errlog(errors.New("有覆盖写入"))
					}
					ch <- nil
					return
				}
			}
		}
	}()

	go func() { // 心跳(进度包)
		for flag {
			time.Sleep(time.Second * 5)
			if err = r.rS3FFFFF0008(rec.Shche()); e.Errlog(err) {
				ch <- err
				return
			}
		}
	}()
	go func() { // 实时速度更新
		for flag {
			r.Speed = 2 * int(r.cspeed)
			r.cspeed = 0
			time.Sleep(time.Millisecond * 500)
		}
	}()

	go func() { // 接收数据包
		go func() { // 设置deadline
			for flag {
				if err = r.conn.SetReadDeadline(time.Now().Add(time.Second * 75)); e.Errlog(err) {
					ch <- err
					return
				}
				time.Sleep(time.Minute)
			}
		}()

		for flag { // 数据包读取
			da = make([]byte, 1500)
			if l, err = r.conn.Read(da); e.Errlog(err) {
				ch <- err
				return
			}
			if dl, bias, tend, err = packet.ParseDataPacket(da[:l], r.key); err == nil {
				if tend && !end {
					fmt.Println("---------------------------收到了结束包-----------------------")
					end = tend
				}
				if bias < 0x3FFFFF0000 {
					if err = w.WriteFile(da[:dl], bias, end); e.Errlog(err) {
						ch <- err
					}

					rec.Add(bias, bias+dl-1) //记录
					if r.forwardBias < bias {
						r.forwardBias = bias
					}
					r.cspeed += dl

				} else {
					fmt.Println("意外偏置", bias)
				}
			} else {
				e.Errlog(err)
			}
		}
	}()

	select {
	case err = <-ch:
		flag = false
		return err
	}
}

var count int = 0

// newSpeed 返回新速度
//  累加，呈指数或线性增加，累减或未达标、变为当前速度
func (r *Receiver) newSpeed() int {

	var ns int
	var thisSpeed = r.Speed
	if thisSpeed == 0 {
		thisSpeed = 5120
	}

	if r.speedsRecorder[r.feedbackLength-1]-thisSpeed == 0 || r.speedsRecorder[r.feedbackLength-1]/(r.speedsRecorder[r.feedbackLength-1]-thisSpeed) > 17 || thisSpeed/(r.Speed-r.speedsRecorder[r.feedbackLength-1]) > 17 { // 达到预期
		// 检测是否累加
		var add bool = true
		for i := 1; i < r.feedbackLength; i++ {
			if add && r.speedsRecorder[i-1] > r.speedsRecorder[i] {
				add = false
			}
		}
		if add {
			// ns = thisSpeed + 2*(r.speedsRecorder[r.feedbackLength-1]-r.speedsRecorder[r.feedbackLength-2])
			ns = thisSpeed + thisSpeed/2

		} else {
			ns = thisSpeed + 10240

		}
	} else { // 未达预期
		ns = thisSpeed + 1024
	}
	r.speedsRecorder = append(r.speedsRecorder[1:], ns)

	return ns
}

//  参数true代表速度增加、即速度变量为正，否则为负，当此轮接收到的数据中缺失的数据大于一定值时、参数就会为false。  传入参数的正负与之前n个参数相比较，如果同号则步长变为2倍。否则以min step的步长增或减(表明此时传输速度与实际带宽相当了)。
//
//  初始化时：最开始的前n个始终记录为0、速度不变(不考虑传输参数的正负)；之后将按照上面的规则进行变换；无论初始速度与实际速度是过大还是过小，都会以指数倍率控制传输速度到相当的速度。
