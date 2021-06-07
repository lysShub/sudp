package sudp

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/lysShub/sudp/speed"

	"github.com/lysShub/sudp/internal/com"
	"github.com/lysShub/sudp/internal/packet"

	"github.com/lysShub/e"
)

type sudp struct {
	Laddr         *net.UDPAddr  // 本地地址
	Raddr         *net.UDPAddr  // 目的地地址
	Encrypt       bool          // 是否数据加密, 默认加密
	MTU           int           // MTU, SUDP包大小、包括包头, 上行/下行不相同, 默认1372
	TimeOut       time.Duration // 数据包超时时间
	Path          string        // 路径, 发送方为发送文件(夹), 接受方位存放路径
	Schedule      int64         // 文件中完整传输的进度
	FileSize      int64         // 当前传输文件大小
	FileName      string        // 当前传输文件名
	Speed         int           // 当前传输速度 B/s
	TansportTotal int64         // 收/发总数据
	Start         time.Time     // 开始传输时间 UTC

	conn      *net.UDPConn
	key       []byte        // 传输密钥, 不数据加密时为空
	secureKey []byte        // 安全加密密钥
	moreDelay time.Duration //多余延时
}

/* 目前还无法传输文件夹, 参见 transfer.go:335 */

var Version uint8 = 0b00000001
var err error

type Read struct {
	sudp

	s *speed.Speed
}
type Write struct {
	sudp
	speed int // 接收到的控制速度
	ds    int // 一个周期内发送的数据包数(一个周期50ms)
}

// NewRead
func NewRead(f func(r *Read) *Read) (*Read, error) {
	var h = new(Read)
	h.MTU = 1372
	h.TimeOut = time.Second * 4
	h.Laddr = &net.UDPAddr{IP: nil, Port: 19986}
	h.moreDelay = moreDalay()
	h = f(h)

	if h.MTU < 500 || h.MTU > 65500 {
		return nil, errors.New("invalid MTU")
	}
	if h.Raddr == nil {
		return nil, errors.New("not set Raddr")
	}

	return h, nil
}

// Read 接收数据
func (r *Read) Read(requestBody []byte) error {

	if err = r.开始握手(requestBody); e.Errlog(err) {
		return err
	}
	r.Start = time.Now()
	var wait bool = false

	// 握手成功 开始传输
	r.s = speed.New(func(s *speed.Speed) *speed.Speed { return s })
	for {
		if name, fs, end, err := r.接收文件信息包或结束包(); e.Errlog(err) {
			return err
		} else {
			wait = false
			if end {
				return nil

			} else {
				r.FileSize = fs
				r.FileName = name

				fmt.Println("储存路径", r.Path+`/`+name)
				if fh, err := openFile(r.Path + `/` + name); e.Errlog(err) {
					return err
				} else {
					if err = r.ReceiveData(fh, fs); e.Errlog(err) { // 读取文件数据
						fh.Close()
						return err
					} else {
						fh.Close()
						if da, err := r.文件结束包数据(); e.Errlog(err) {
							return err
						} else {
							wait = true
							go func() {
								for wait {
									if _, err = r.conn.Write(da); e.Errlog(err) { // 发送文件结束包
										return
									}
									time.Sleep(time.Millisecond * 10)
								}
							}()
						}

					}
				}
			}
		}
	}

}

// NewWrite
func NewWrite(f func(w *Write) *Write) (*Write, error) {
	var h = new(Write)
	h.Encrypt = true
	h.MTU = 1372
	h.TimeOut = time.Second
	h.Laddr = &net.UDPAddr{IP: nil, Port: 19986}
	h.moreDelay = moreDalay()
	// 无需设置Raddr
	h = f(h)
	if h.Path == "" {
		return nil, errors.New("not set Path")
	}
	if h.MTU < 500 || h.MTU > 65500 {
		return nil, errors.New("invalid MTU")
	}
	return h, nil
}

// Write 发送数据
//  会阻塞直到收到接收方请求
func (w *Write) Write(f func(requestBody []byte) bool) error {

	if ifs, basePath, outFiles, err := com.GetFloderInfo(w.Path); err != nil || len(outFiles) != 0 {
		if e.Errlog(err) {
			return err
		} else {
			return errors.New("read " + outFiles[0] + " Access is denied.")
		}
	} else {

		// 握手
		if err = w.等待握手(f); e.Errlog(err) {
			return err
		}
		w.Start = time.Now()

		var fh *os.File
		for i, n := range ifs.N {
			w.FileSize = ifs.S[i]
			w.FileName = ifs.N[i]
			if w.FileSize == 0 {
				fmt.Println("文件", n, "大小为0, 将不被传输")
				continue
			}

			if fh, err = os.Open(basePath + `/` + n); e.Errlog(err) {
				return err
			}

			if err = w.发送文件信息包并接收开始包(n, ifs.S[i]); e.Errlog(err) {
				return err
			}
			// fmt.Println("发送文件数据", n)

			var sch int64
			if sch, err = w.SendData(fh, ifs.S[i]); e.Errlog(err) {
				return err
			}
			if sch != ifs.S[i] {
				return nil
			}
		}

		// 发送任务结束包
		if da, err := packet.SecureEncrypt(nil, w.secureKey); e.Errlog(err) {
			return err
		} else {
			if da, _, _, err = packet.PackagePacket(da, 0x3FFFFFFF00, w.key, false); e.Errlog(err) {
				return err
			}
			for i := 0; i < 27; i++ {
				if _, err = w.conn.Write(da); e.Errlog(err) {
					return err
				}
			}
		}
	}

	return nil
}
