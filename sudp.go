package sudp

import (
	"errors"
	"net"
	"os"
	"time"

	"gitee.com/lysshub/sudp/internal/com"
	"gitee.com/lysshub/sudp/internal/packet"

	"github.com/lysShub/e"
)

type sudp struct {
	Encrypt  bool          // 是否加密, 默认加密
	MTU      int           // MTU, SUDP包大小、包括包头, 上行/下行不相同, 默认1372
	TimeOut  time.Duration // 数据包超时时间
	Path     string        // 路径, 发送方为发送文件(夹), 接受方位存放路径
	Schedule int64         // 已传输进度
	Speed    int           // 当前传输速度 B/s
	Laddr    *net.UDPAddr  //
	Raddr    *net.UDPAddr  //

	conn       *net.UDPConn
	key        []byte        // 传输密钥, 可能为nil
	controlKey []byte        // 必须被设置, 用于加密控制包的数据
	moreDelay  time.Duration //多余延时
}

var Version uint8 = 0b00000001
var err error

type Read struct {
	sudp
	newSpeed int
	total    int64 // 记录总接收
}
type Write struct {
	sudp
	// ts time.Duration //
	ds    int   // 一个周期内发送的数据包数(一个周期50ms)
	total int64 // 记录总发送
}

// NewRead
func NewRead(f func(r *Read) *Read) (*Read, error) {
	var y = new(Read)
	y.MTU = 1372
	y.TimeOut = time.Second
	y.Laddr = &net.UDPAddr{IP: nil, Port: 19986}
	y = f(y)

	if y.MTU < 500 || y.MTU > 1500 {
		return nil, errors.New("invalid MTU")
	}
	if y.Raddr == nil {
		return nil, errors.New("not set Raddr")
	}
	return y, nil
}

// Read 接收数据
func (r *Read) Read(requestBody []byte) error {

	if err = r.sendHandshake(requestBody); e.Errlog(err) {
		return err
	}
	for {
		if name, fs, end, err := r.receiverFileInfoOrEndPacket(); e.Errlog(err) {
			return err
		} else {
			if end {
				e.Errlog(errors.New("任务结束"))
				return nil

			} else {
				if fh, err := openFile(r.Path + `/` + name); e.Errlog(err) {
					return err
				} else {
					if err = r.receiveData(fh, fs); e.Errlog(err) { // 读取文件数据
						fh.Close()
						return err

					} else {
						fh.Close()
						if err = r.sendFileEndPacket(); e.Errlog(err) {
							return err
						}
					}
				}
			}
		}

	}

	return nil
}

// NewWrite
func NewWrite(f func(r *Write) *Write) (*Write, error) {
	var y = new(Write)
	y.Encrypt = true
	y.MTU = 1372
	y.TimeOut = time.Second
	y.Laddr = &net.UDPAddr{IP: nil, Port: 19986}
	// 无需设置Raddr
	y.moreDelay = 0 //moreDalay()
	y = f(y)
	if y.Path == "" {
		return nil, errors.New("not set Path")
	}
	if y.MTU < 500 || y.MTU > 1500 {
		return nil, errors.New("invalid MTU")
	}
	return y, nil
}

// Write 发送数据
//  会阻塞直到收到接收方请求
func (w Write) Write(f func(requestBody []byte) bool) error {

	if ifs, basePath, outFiles, err := com.GetFloderInfo(w.Path); err != nil || len(outFiles) != 0 {
		if e.Errlog(err) {
			return err
		} else {
			return errors.New("read " + outFiles[0] + " Access is denied.")
		}
	} else {

		// 握手
		if err = w.receiveHandshake(f); e.Errlog(err) {
			return err
		}

		// ---------------发送数据-------------------- //
		var fh *os.File
		for i, n := range ifs.N {
			if fh, err = os.Open(basePath + `/` + n); e.Errlog(err) {
				return err
			}

			if err = w.sendFileInfoAndReceiveStartPacket(n, ifs.S[i]); e.Errlog(err) {
				return err
			}

			var sch int64
			if sch, err = w.sendData(fh, ifs.S[i]); e.Errlog(err) {
				return err
			}
			if sch != ifs.S[i] {
				return nil
			}
		}

		// 发送任务结束包
		if da, err := packet.SecureEncrypt(nil, w.controlKey); e.Errlog(err) {
			return err
		} else {
			if da, _, _, err = packet.PackagePacket(da, 0x3FFFFFFF00, w.key, false); e.Errlog(err) {
				return err
			}
			for i := 0; i < 10; i++ {
				if _, err = w.conn.Write(da); e.Errlog(err) {
					return err
				}
			}
		}
	}

	return nil
}
