package sudp

/* 数据发送 */

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sudp/internal/file"
	"sudp/internal/packet"
	"sudp/internal/recorder"
	"sudp/internal/strategy"
	"time"

	"github.com/lysShub/e"
)

// sendData 发送数据
func (w *Write) sendData(fh *os.File, fileSize int64) (int64, error) {

	r := new(file.Rd) // 读取器
	r.Fh = fh

	var errCh chan error = make(chan error, 2)
	var endCh chan int64 = make(chan int64, 1)
	var flag bool = true
	defer func() { flag = false }()
	var ts time.Duration = time.Millisecond * 10 // 数据包间隙暂停时间

	// 接收
	go func() {
		var (
			da       []byte
			bias, dl int64
			l        int
		)
		go func() {
			for flag {
				if err = w.conn.SetReadDeadline(time.Now().Add(time.Second * 16)); e.Errlog(err) {
					errCh <- err
				}
				time.Sleep(time.Second * 15)
			}
		}()

		for flag {

			da = make([]byte, 1500)
			if l, err = w.conn.Read(da); err != nil {
				if strings.Contains(err.Error(), "closed") {
					errCh <- errors.New("broken, no data for more than 15 seconds")
					return
				} else {
					errCh <- err
					return
				}
			} else {
				if dl, bias, _, err = packet.ParseDataPacket(da[:l], w.key); err == nil {

					if bias == 0x3FFFFF0004 { // 文件重发包
						fmt.Println("接收到文件重发包")

						ts = time.Second // 优先处理重发数据, 暂停主进程发送
						if err = w.receiveResendDataPacket(da[:dl], r); e.Errlog(err) {
							errCh <- err
							return
						}
						if w.Speed > 0 {
							ts = time.Duration(1e9 * w.MTU / w.Speed)
						} else {
							ts = time.Millisecond * 10
						}

					} else if bias == 0x3FFFFF0008 { // 文件进度包

						w.Schedule = int64(da[0])<<32 + int64(da[1])<<24 + int64(da[2])<<16 + int64(da[3])<<8 + int64(da[4])
						fmt.Println("接收到文件进度包", w.Schedule)

					} else if bias == 0x3FFFFF00FF { // 文件结束包

						fmt.Println("收到文件结束包")
						endCh <- fileSize
						return

					} else if bias == 0x3FFFFF0010 {

						fmt.Println("收到速度控制包", w.Speed, int(da[0])<<24+int(da[1])<<16+int(da[2])<<8+int(da[3]))

						w.Speed = int(da[0])<<24 + int(da[1])<<16 + int(da[2])<<8 + int(da[3])

					} else {
						fmt.Println("意外偏置", bias)
					}
				} else {
					e.Errlog(err)
				}
			}
		}
	}()

	// ((主进程)发送
	go func() {
		var d []byte
		var sEnd bool
		var dl, bias int64
		go func() { // 更新ts
			for flag {
				if w.Speed > 0 {
					ts = time.Duration(1e9 * w.MTU / w.Speed)
				} else {
					ts = time.Millisecond * 10
				}
				time.Sleep(time.Millisecond * 10)
			}
		}()

		for bias = int64(0); bias < fileSize; {

			d = make([]byte, w.MTU-9, w.MTU+8)
			if d, dl, sEnd, err = r.ReadFile(d, bias, w.key); e.Errlog(err) {
				errCh <- err
				return
			}
			if _, err = w.conn.Write(d); e.Errlog(err) {
				errCh <- err
				return
			}
			bias = bias + dl
			time.Sleep(ts)

			if sEnd { // 最后数据包必达
				for {
					time.Sleep(time.Millisecond * 500)
					if _, err = w.conn.Write(d); e.Errlog(err) {
						errCh <- err
						return
					}
				}
			}
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
	var counter int64 = 0 // 记录一段时间收到的数据

	go func() { // 速度
		for flag { // 周期
			time.Sleep(strategy.SpeedTime)

			// 速度控制
			n := strategy.NewSpeed(r.Speed)
			fmt.Println("新速度", n, r.Speed)
			if err = r.sendSpeedControlPacket(n); e.Errlog(err) {
				fmt.Println(err)
			}
		}
	}()

	go func() { // 重发
		for flag {
			time.Sleep(strategy.ResendTime)

			if re := rec.Owe(0); len(re) > 0 || end {
				if err = r.sendResendDataPacket(re); e.Errlog(err) {
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
			if err = r.sendSchedulPacket(rec.Shche()); e.Errlog(err) {
				ch <- err
				return
			}
		}
	}()
	go func() { // 速度更新
		for flag {
			r.Speed = 5 * int(counter)
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
					counter += dl

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

/* ------------------------------- */

func (r *Read) sendSpeedControlPacket(ns int) error {
	var da []byte = []byte{uint8(ns >> 24), uint8(ns >> 16), uint8(ns >> 8), uint8(ns)}
	if da, _, _, err = packet.PackageDataPacket(da, 0x3FFFFF0010, r.key, false); e.Errlog(err) {
		return err
	}
	if _, err = r.conn.Write(da); e.Errlog(err) {
		return err
	}
	return nil
}

func (r *Read) sendResendDataPacket(ownRec [][2]int64) error {
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

func (r *Read) sendSchedulPacket(sch int64) error {
	var da []byte
	if da, _, _, err = packet.PackageDataPacket([]byte{uint8(sch >> 32), uint8(sch >> 24), uint8(sch >> 16), uint8(sch >> 8), uint8(sch)}, 0x3FFFFF0008, r.key, false); e.Errlog(err) {
		return err
	}
	if _, err = r.conn.Write(da); e.Errlog(err) {
		return err
	}
	return nil
}

func (r *Read) receiverFileInfoOrEndPacket() (string, int64, bool, error) {
	var da []byte
	var l int

	time.AfterFunc(r.TimeOut, func() {
		fmt.Println("关闭了r.conn.Close()")
		r.conn.Close()
	})
	for {
		da = make([]byte, 1500)
		if l, err = r.conn.Read(da); err != nil {
			if strings.Contains(err.Error(), "closed") {
				return "", 0, false, errors.New("timeout")
			} else if e.Errlog(err) {
				return "", 0, false, err
			}
		}
		if dl, bias, _, err := packet.ParseDataPacket(da[:l], r.key); err == nil {

			if bias == 0x3FFFFF0001 { // 文件信息

				return string(da[5:dl]), int64(da[0])<<32 + int64(da[1])<<24 + int64(da[2])<<16 + int64(da[3])<<8 + int64(da[4]), false, nil

			} else if bias == 0x3FFFFFFF00 { // 任务结束包
				return "", 0, true, nil
			}
		}
	}
	return "", 0, true, nil
}

func (w *Write) sendFileInfoAndReceiveStartPacket(name string, fs int64) error {
	var sda, rda []byte = []byte{uint8(fs >> 32), uint8(fs >> 24), uint8(fs >> 16), uint8(fs >> 8), uint8(fs)}, nil
	sda = append(sda, []byte(name)...)

	if sda, _, _, err = packet.PackageDataPacket(sda, 0x3FFFFF0001, w.key, false); e.Errlog(err) {
		return err
	}

	var flag bool = true
	go func() {
		for flag {
			if _, err = w.conn.Write(sda); e.Errlog(err) {
				return
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()
	var step int = 0
	time.AfterFunc(w.TimeOut*2, func() {
		if step < 1 {
			fmt.Println("关闭了w.conn")
			w.conn.Close()
		}
	})
	for {
		rda = make([]byte, 1500)
		if l, err := w.conn.Read(rda); err != nil {
			if strings.Contains(err.Error(), "closed") {
				return errors.New("timeout")
			}
			return err
		} else {
			if _, bias, _, err := packet.ParseDataPacket(rda[:l], w.key); e.Errlog(err) {
				continue
			} else if bias == 0x3FFFFF0002 {
				step = 1
				return nil
			}
		}
	}
}

func (w *Write) receiveResendDataPacket(da []byte, r *file.Rd) error {

	var sb, eb int64
	var d []byte
	ts := time.Duration(1e9 * w.MTU / w.Speed)

	for i := 9; i <= len(da); i = i + 10 {

		sb = int64(da[i-9])<<32 + int64(da[i-8])<<24 + int64(da[i-7])<<16 + int64(da[i-6])<<8 + int64(da[i-5])
		eb = int64(da[i-4])<<32 + int64(da[i-3])<<24 + int64(da[i-2])<<16 + int64(da[i-1])<<8 + int64(da[i-0])

		for i := sb; i <= eb; i = i + int64(w.MTU) {
			if int64(w.MTU)+i-1 > eb {
				d = make([]byte, eb-i+1)
			} else {
				d = make([]byte, w.MTU)
			}
			if d, _, _, err = r.ReadFile(d, i, w.key); e.Errlog(err) {
				return err
			}
			if _, err = w.conn.Write(d); e.Errlog(err) {
				return err
			}
			time.Sleep(ts)
		}
	}

	return nil
}

func openFile(path string) (*os.File, error) {
	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(filepath.Dir(path), 0666); err != nil {
			return nil, err
		}
		if fh, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666); err != nil {
			return nil, err
		}
	}
	return fh, nil
}
