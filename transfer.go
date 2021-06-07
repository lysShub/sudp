package sudp

/* 数据发送 */

import (
	"errors"
	"fmt"
	"os"
	"time"

	"sudp/internal/file"
	"sudp/internal/packet"
	"sudp/internal/recorder"
	"sudp/internal/strategy"

	"github.com/lysShub/e"
)

// sendData 发送数据
func (w *Write) sendData(fh *os.File, fileSize int64) (int64, error) {

	r := new(file.Rd) // 读取器
	r.Fh = fh

	var errCh chan error = make(chan error, 2)   // 错误通知管道
	var endCh chan int64 = make(chan int64, 1)   // 结束通知管道
	var senCh chan int64 = make(chan int64, 256) // 发送管道

	var flag bool = true // 结束使能, 用于退出协程
	defer func() { flag = false }()
	w.ds = 5                 // 最小发送速度为5*20*MTU/s
	var resFlag bool = false // 重发时, 控制主发送进程停止, 即优先处理重发数据

	// 接收
	go func() {
		var (
			da               []byte
			bias, dl, eb, sb int64
			l                int
			keep             bool = false
		)
		go func() {
			for flag { // 15s超时(最多30s)
				time.Sleep(time.Second * 15)
				if !keep {
					errCh <- errors.New("broken, no data for more than 15 seconds")
					return
				}
				keep = false
			}
		}()

		for flag {
			da = make([]byte, 1500)
			if l, err = w.conn.Read(da); err != nil {
				// read: connection refused 表示对方关闭, UDP是无连接的, 这由于ICMP通知的
				errCh <- err
				return
			} else {
				if dl, bias, _, err = packet.ParsePacket(da[:l], w.key); err == nil {

					if bias == 0x3FFFFF0004 { // 文件重发包
						if da, err = packet.SecureDecrypt(da[:dl], w.controlKey); e.Errlog(err) {
							errCh <- err
							return
						} else {
							keep = true

							resFlag = true
							for i := 9; i <= len(da); i = i + 10 {
								sb = int64(da[i-9])<<32 + int64(da[i-8])<<24 + int64(da[i-7])<<16 + int64(da[i-6])<<8 + int64(da[i-5])
								eb = int64(da[i-4])<<32 + int64(da[i-3])<<24 + int64(da[i-2])<<16 + int64(da[i-1])<<8 + int64(da[i-0])

								for j := sb; j <= eb; j = j + int64(w.MTU-9) {
									senCh <- j
								}
							}
							resFlag = false

						}

					} else if bias == 0x3FFFFF0008 { // 文件进度包
						if da, err = packet.SecureDecrypt(da[:dl], w.controlKey); err == nil && len(da) == 5 {
							keep = true
							w.Schedule = int64(da[0])<<32 + int64(da[1])<<24 + int64(da[2])<<16 + int64(da[3])<<8 + int64(da[4])

						} else {
							e.Errlog(err)
						}

					} else if bias == 0x3FFFFF00FF { // 文件结束包

						if _, err = packet.SecureDecrypt(da[:dl], w.controlKey); err == nil {
							endCh <- fileSize
							return
						} else {
							e.Errlog(err)
						}

					} else if bias == 0x3FFFFF0010 { // 速度控制包

						if da, err = packet.SecureDecrypt(da[:dl], w.controlKey); err == nil && len(da) == 4 {
							w.Speed = int(da[0])<<24 + int(da[1])<<16 + int(da[2])<<8 + int(da[3])

							// fmt.Println("收到速度控制包", w.Speed, int(da[0])<<24+int(da[1])<<16+int(da[2])<<8+int(da[3]))
						} else {
							e.Errlog(err)
						}

					} else {
					}
				} else {
					e.Errlog(err)
				}
			}
		}
	}()

	// 更新ts
	go func() {
		for flag {
			if w.Speed > 0 {
				// w.ts = time.Duration(10 * 1e9 * w.MTU / w.Speed) //- 20000
				w.ds = w.Speed >> 4 / w.MTU
				if w.ds < 5 {
					w.ds = 5
				}
			} else {
				w.ds = 5
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()
	go func() {
		for {
			time.Sleep(time.Second * 2)
			fmt.Println(w.Speed)
		}
	}()

	// 发送
	go func() {
		var d []byte
		var count, n int
		var bf time.Time = time.Now()
		for flag {

			d = make([]byte, w.MTU-9, w.MTU+16)
			if d, _, _, err = r.ReadFile(d, <-senCh, w.key); e.Errlog(err) {
				errCh <- err
				return
			}
			if n, err = w.conn.Write(d); e.Errlog(err) {
				errCh <- err
				return
			}

			w.total = w.total + int64(n)

			count++
			if count > w.ds {
				time.Sleep(62500000 - w.moreDelay - time.Since(bf))
				bf = time.Now()
				count = 0
			}
		}
	}()

	// 主进程通知发送
	go func() {
		var bias int64
		for bias = int64(0); bias < fileSize; {
			if !resFlag {
				senCh <- bias
				bias = bias + int64(w.MTU-9)
			} else {
				time.Sleep(time.Millisecond * 5) // 优先处理重发
			}
		}
		// 最后包必达
		for flag {
			time.Sleep(time.Millisecond * 500)
			senCh <- fileSize - 100
		}
	}()

	select {
	case err = <-errCh: // 出错
		return w.Schedule, err
	case r := <-endCh: // 结束
		return r, nil
	}
}

// receiveData 接收数据
func (r *Read) receiveData(fh *os.File, fs int64) error {
	w := new(file.Wt) // 写入器
	w.Fh = fh

	rec := new(recorder.Recorder) // 记录器
	defer rec.End()
	rec.NewRecorder()

	var da []byte = make([]byte, 1500)

	var end, tend, flag = false, false, true // 接收到最后包, _ , 结束传输
	defer func() { flag = false }()

	var ch chan error = make(chan error)
	var counter int = 0 // 记录一段时间收到的数据

	go func() { // 速度
		for flag { // 周期
			time.Sleep(strategy.SpeedTime)

			// 速度控制
			n := strategy.NewSpeed(r.Speed)
			r.newSpeed = n
			if err = r.sendSpeedControlPacket(n); e.Errlog(err) {
				// fmt.Println(err)
			}
		}
	}()

	go func() { // 重发

		var re [][2]int64
		for flag {
			if !end {
				time.Sleep(strategy.ResendTime)
				if re = rec.Owe(); len(re) > 0 {
					if err = r.sendResendDataPacket(re); e.Errlog(err) {
						ch <- err
						return
					}
				}

			} else { // 收到最后包, 只剩重发, 改变重发策略

				rr := rec.OweAll()
				for _, re = range rr {
					if err = r.sendResendDataPacket(re); e.Errlog(err) {
						ch <- err
						return
					}
					time.Sleep(time.Duration(1e9*100*r.MTU/r.newSpeed) - r.moreDelay)
				}
				time.Sleep(strategy.ResendTime)

			}

			if end && rec.Blocks() == 1 {
				ch <- nil
				return
			}
		}
	}()

	go func() { // 心跳(进度包)
		for flag {
			time.Sleep(time.Second)
			r.Schedule = rec.Shche()
			if err = r.sendSchedulPacket(r.Schedule); e.Errlog(err) {
				ch <- err
				return
			}
		}
	}()
	go func() { // 速度更新
		for flag {
			r.Speed = 5 * counter
			counter = 0
			time.Sleep(time.Millisecond * 200)
		}
	}()

	go func() { // 接收数据包

		var (
			l        int = 0
			dl, bias int64
		)
		for flag {
			da = make([]byte, 1500)
			if l, err = r.conn.Read(da); e.Errlog(err) {
				ch <- err
				return
			}
			if dl, bias, tend, err = packet.ParsePacket(da[:l], r.key); err == nil {
				if tend && !end {
					end = tend
				}
				if bias < 0x3FFFFF0000 {
					if err = w.WriteFile(da[:dl], bias, end); e.Errlog(err) {
						// write XXX: file already closed 由于传输已经完成, 还有“迟到的数据包”
						ch <- err
					}
					rec.Add(bias, bias+dl-1) //记录
					counter += int(dl)
					r.total += dl

				} else {
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

/* ------------------------------- */
