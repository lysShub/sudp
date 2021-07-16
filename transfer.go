package sudp

/* 数据发送 */

import (
	"fmt"
	"os"
	"time"

	"github.com/lysShub/sudp/internal/file"
	"github.com/lysShub/sudp/internal/packet"
	"github.com/lysShub/sudp/internal/recorder"

	"github.com/lysShub/e"
)

// SendData 发送数据
func (w *Write) SendData(fh *os.File, fileSize int64) (int64, error) {

	r, err := file.NewRead(fh)
	if err != nil {
		return 0, err
	}

	var errCh chan error = make(chan error, 2)    // 错误通知管道
	var endCh chan int64 = make(chan int64, 1)    // 结束通知管道
	var senCh chan int64 = make(chan int64, 2048) // 发送管道

	var flag bool = true // 结束使能, 用于退出协程
	defer func() { flag = false }()
	w.ds = 5 // 最小发送速度为5*20*MTU/s
	// var resFlag bool = false // 重发时, 控制主发送进程停止, 即优先处理重发数据

	// 接收
	go func() {
		var (
			da               []byte
			bias, dl, eb, sb int64
			l                int
		)

		for flag {
			da = make([]byte, w.MTU+16)
			if l, err = w.conn.Read(da); err != nil {
				// read: connection refused 表示对方关闭, UDP是无连接的, 这由于ICMP通知的
				errCh <- err
				return
			} else {
				if dl, bias, _, err = packet.ParsePacket(da[:l], w.key); err == nil {

					if bias == 0x3FFFFF0004 { // 文件重发包
						if da, err = packet.SecureDecrypt(da[:dl], w.secureKey); e.Errlog(err) {
							errCh <- err
							return
						} else {

							for i := 9; i <= len(da); i = i + 10 {
								sb = int64(da[i-9])<<32 + int64(da[i-8])<<24 + int64(da[i-7])<<16 + int64(da[i-6])<<8 + int64(da[i-5])
								eb = int64(da[i-4])<<32 + int64(da[i-3])<<24 + int64(da[i-2])<<16 + int64(da[i-1])<<8 + int64(da[i-0])

								for j := sb; j <= eb; j = j + int64(w.MTU-9) {
									senCh <- j
								}
							}

						}

					} else if bias == 0x3FFFFF0008 { // 文件进度包
						if da, err = packet.SecureDecrypt(da[:dl], w.secureKey); err == nil && len(da) == 5 {
							w.Schedule = int64(da[0])<<32 + int64(da[1])<<24 + int64(da[2])<<16 + int64(da[3])<<8 + int64(da[4])

						} else {
							e.Errlog(err)
						}

					} else if bias == 0x3FFFFF00FF { // 文件结束包

						if _, err = packet.SecureDecrypt(da[:dl], w.secureKey); err == nil {
							endCh <- fileSize
							return
						} else {
							e.Errlog(err)
						}

					} else if bias == 0x3FFFFF0010 { // 速度控制包

						if da, err = packet.SecureDecrypt(da[:dl], w.secureKey); err == nil && len(da) == 4 {
							w.speed = int(da[0])<<24 + int(da[1])<<16 + int(da[2])<<8 + int(da[3])

						} else {
							e.Errlog(err)
						}

					}
				} else {
					e.Errlog(err)
				}
			}
		}
	}()

	// 更新ds
	go func() {
		for flag {
			if w.speed > 0 {
				w.ds = w.speed >> 4 / w.MTU
				if w.ds < 5 {
					w.ds = 5
				}
			} else {
				w.ds = 5
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()

	// 更新实际发送速度
	var scount int64 = 0
	go func() {
		for flag {
			time.Sleep(time.Millisecond*200 - w.moreDelay)
			w.Speed = int(scount) * 5
			scount = 0
		}
	}()

	// 发送
	go func() {
		var d []byte
		var count int
		var ol int64
		var bf time.Time = time.Now()
		for flag {

			d = make([]byte, w.MTU-9, w.MTU+16)
			if d, ol, _, err = r.ReadFile(d, <-senCh, w.key); e.Errlog(err) {
				errCh <- err
				return
			}
			if _, err = w.conn.Write(d); e.Errlog(err) {
				errCh <- err
				return
			}

			w.TansportTotal += ol
			scount += ol
			count++

			if count > w.ds {
				time.Sleep(62500000 - w.moreDelay - time.Since(bf))
				bf = time.Now()
				count = 0
			}
		}
	}()

	// 主进程发送
	go func() {
		var bias int64
		for bias = int64(0); bias < fileSize; {
			senCh <- bias
			bias = bias + int64(w.MTU-9)
			// if !resFlag {
			// 	senCh <- bias
			// 	bias = bias + int64(w.MTU-9)
			// } else {
			// 	time.Sleep(time.Millisecond * 5)
			// }
		}

		fmt.Println("主进程发送完成")
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

// ReceiveData 接收数据
func (r *Read) ReceiveData(fh *os.File, fs int64) error {
	w := new(file.Wt) // 写入器
	w.Fh = fh

	rec := new(recorder.Recorder) // 记录器
	defer rec.End()
	rec.NewRecorder()

	var end, tend, flag = false, false, true // 接收到最后包, _ , 结束传输

	var ch chan error = make(chan error)
	var counter int = 0 // 记录一段时间收到的数据

	// 速度控制
	go func() {
		for flag {
			time.Sleep(r.s.SpeedPeriod)

			e.Errlog(r.发送速度控制包(r.s.NewSpeed(r.Speed)))
		}
	}()

	// 重发
	go func() {
		var re [][2]int64
		for flag {
			if !end {
				time.Sleep(r.s.ResendPeriod)
				if re = rec.Owe(); len(re) > 0 {
					if err = r.文件重发包(re); e.Errlog(err) {
						ch <- err
						return
					}
				}

			} else { // 收到最后包, 只剩重发, 改变重发策略
				rr := rec.OweAll()

				// 寻找异常
				if end && len(rr) == 0 {
					fmt.Println("发现bug")
					fmt.Println(rec.Expose())
					ch <- err
					return
				}

				for _, re = range rr {
					if err = r.文件重发包(re); e.Errlog(err) {
						ch <- err
						return
					}
					// time.Sleep(time.Duration(1e9*100*r.MTU/r.newSpeed) - r.moreDelay)
				}

				time.Sleep(r.s.ResendPeriod)

			}

			if end && rec.Complete(fs) { // 传输完成 判定方式有bug
				ch <- nil
				return
			}
		}
	}()

	// 进度包(心跳)
	go func() {
		for flag {
			time.Sleep(time.Second)
			r.Schedule = rec.Shche()
			if err = r.sendSchedulPacket(r.Schedule); e.Errlog(err) {
				ch <- err
				return
			}
		}
	}()

	// 更新本地实时速度
	go func() {
		for flag {
			r.Speed = 5 * counter
			counter = 0
			time.Sleep(time.Millisecond*200 - r.moreDelay)
		}
	}()

	// 接收数据包
	go func() {
		var (
			l        int = 0
			dl, bias int64
		)
		var da []byte = make([]byte, r.MTU+16)

		for flag {
			da = make([]byte, r.MTU+16)
			if l, err = r.conn.Read(da); e.Errlog(err) {
				ch <- err
				return
			}
			if dl, bias, tend, err = packet.ParsePacket(da[:l], r.key); err == nil {
				r.TansportTotal += dl

				if tend && !end {
					// fmt.Println("主进程传输完成", r.TansportTotal)
					end = tend
				}
				if bias < 0x3FFFFF0000 && flag {
					if err = w.WriteFile(da[:dl], bias, end); e.Errlog(err) {
						ch <- err
					}
					rec.Add(bias, bias+dl-1) // 更新记录
					counter += int(dl)

				}
			} else {
				e.Errlog(err)
			}
		}

	}()

	err = <-ch
	flag = false
	return err

}
