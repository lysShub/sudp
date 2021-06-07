package utils

import (
	"btest/com"
	"btest/setting"
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"btest/sudp"
)

var err error

func UpLoad(path string) error {
	var (
		end bool       = false
		ch  chan error = make(chan error)
	)

	var s *setting.S
	if s, err = setting.Read(); err != nil {
		s = setting.DefaultConfig()
	}

	w, err := sudp.NewWrite(func(w *sudp.Write) *sudp.Write {
		w.Laddr = &net.UDPAddr{IP: s.LIP, Port: s.LPort}
		w.Path = path
		w.MTU = s.MTU
		w.Encrypt = s.Crypt
		return w
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 传输数据
	go func() {
		if err = w.Write(func(requestBody []byte) bool {
			if bytes.Equal(requestBody, []byte(s.Auth)) {
				return true
			} else {
				fmt.Println("权鉴不通过")
				return false
			}
		}); err != nil {
			ch <- err
		} else {
			ch <- nil
		}
		end = true
	}()

	// 进度条
	go func() {
		var i int
		for !end {
			if w.FileSize != 0 {
				i = int(w.Schedule * 100 / w.FileSize)
			} else {
				i = 0
			}
			time.Sleep(time.Second)
			fmt.Fprintf(os.Stdout, "%-12s %-2d%% [%s]\r", com.FormatSpeed(w.Speed), i, com.GetS(i, "#")+com.GetS(100-i, " "))
		}
		// fmt.Fprintf(os.Stdout, "%-12s %-2d%% [%s]\r", com.FormatSpeed(w.Speed), i, com.GetS(i, "#")+com.GetS(100-i, " "))
		fmt.Fprintf(os.Stdout, "%-12s %-2d%% [%s]\r", com.FormatSpeed(w.Speed), 100, com.GetS(100, "#")+com.GetS(0, " "))
	}()

	err = <-ch
	time.Sleep(time.Second)
	fmt.Println("")
	fmt.Println("发送总数据:", w.TansportTotal, " Byte", "      用时:", time.Since(w.Start))
	end = true
	return err
}

// DownLoad 下载, rPort为0表示使用默认端口
func DownLoad(rIP net.IP, rPort int) error {

	var s *setting.S
	if s, err = setting.Read(); err != nil {
		s = setting.DefaultConfig()
	}
	if rPort == 0 {
		rPort = s.RPort
	}
	fmt.Println("下载，请求对方端口", rPort)

	r, err := sudp.NewRead(func(r *sudp.Read) *sudp.Read {
		r.Raddr = &net.UDPAddr{IP: rIP, Port: rPort}
		r.Path = s.StorePath
		r.Encrypt = s.Crypt
		r.MTU = s.MTU
		return r
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	var (
		end bool       = false
		ch  chan error = make(chan error)
	)
	go func() {
		if err = r.Read([]byte(s.Auth)); err != nil {
			ch <- err
		} else {
			ch <- nil
		}
	}()

	go func() {
		var i int
		for !end {
			if r.FileSize != 0 {
				i = int(r.Schedule * 100 / r.FileSize)
			} else {
				i = 0
			}

			fmt.Fprintf(os.Stdout, "%-12s %-2d%% [%s]\r", com.FormatSpeed(r.Speed), i, com.GetS(i, "#")+com.GetS(100-i, " "))
			time.Sleep(time.Millisecond * 500)
			// fmt.Println(r.Speed)

		}
		// fmt.Fprintf(os.Stdout, "%-12s %-2d%% [%s]\r", com.FormatSpeed(r.Speed), i, com.GetS(i, "#")+com.GetS(100-i, " "))
		fmt.Fprintf(os.Stdout, "%-12s %-2d%% [%s]\r", com.FormatSpeed(r.Speed), 100, com.GetS(100, "#")+com.GetS(0, " "))
	}()

	err = <-ch
	end = true
	time.Sleep(time.Second)
	fmt.Println("")
	fmt.Println("接收总数据", r.TansportTotal, " Byte", "      用时:", time.Since(r.Start))

	return err
}
