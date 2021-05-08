package tasker

/* 一个传输任务 */

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sudp/internal/com"
	"time"

	"github.com/lysShub/e"
)

// Tasker 代表一个传输任务
//
type Tasker struct {
	Addr         *net.UDPAddr  // 对于接收发是发送方公网地址(不可省略)。对于发送方是自己的内网地址(通常IP为nil、默认端口19986)
	Encrypto     bool          // 是否将加密传输, 仅发送方
	Path         string        // 对于接收方的存储文件夹路径。对于发送方是发送的文件(夹)路径(不可省略)。
	conn         *net.UDPConn  // udp conn
	matchTimeOut time.Duration //

	Sender
	Receiver
}

var err error
var reliable int = 5 // 重发UDP包增加可靠性

// Send 发送一个文件/文件夹
//  fun起权鉴作用, 处理任务请求包的body
func (t *Tasker) Send(fun func(requestBody []byte) bool) error {
	t.sinit()

	if ifs, basePath, outFiles, err := com.GetFloderInfo(t.Path); err != nil || len(outFiles) != 0 {
		if e.Errlog(err) {
			return err
		} else {
			return errors.New("read " + outFiles[0] + " Access is denied.")
		}
	} else {
		/* ----流程---- */
		if err = t.sR3FFFFF0000(t.Addr, fun); e.Errlog(err) {
			return err
		}
		fmt.Println("接收请求包")
		if err = t.sS3FFFFF8000(t.Encrypto); e.Errlog(err) {
			return err
		}
		fmt.Println("发送握手包")
		if pubkey, err := t.sR3FFFFF4000(); e.Errlog(err) {
			return err
		} else {
			fmt.Println("接收确认握手包")
			if err = t.sS3FFFFF2000(pubkey); e.Errlog(err) {
				return err
			}
			fmt.Println("发送认确确认认握手包")
		}
		if err = t.sR3FFFFF1000(); e.Errlog(err) {
			return err
		}
		fmt.Println("接收到任务开始包")

		//  握手成功  //
		fmt.Println("握手成功")

		// ---------------发送数据-------------------- //
		var fh *os.File
		for i, n := range ifs.N {
			fmt.Println("name", n)
			if fh, err = os.Open(basePath + `/` + n); e.Errlog(err) {
				return err
			}

			if err = t.sS3FFFFF0001(n, ifs.S[i]); e.Errlog(err) {
				return err
			}
			fmt.Println("发送文件信息包")

			if err = t.sR3FFFFF0002(); e.Errlog(err) {
				return err
			}
			fmt.Println("接收到文件开始包")

			if s, err := t.sSFileDataPacket(fh, ifs.S[i]); e.Errlog(err) {
				return err
			} else if s != ifs.S[i] {
				fmt.Println("没有传输中止", s, ifs.S[i])
				return nil
			}

		}
		if err = t.sS3FFFFFFF00(); e.Errlog(err) {
			return err
		}
	}
	return nil
}

// Receive 接收一个文件/文件夹
//  basePath是储存路径; requestBody是任务请求包的数据部分
func (t *Tasker) Receive(laddr *net.UDPAddr, requestBody []byte) error {
	t.rinit()

	if err = t.rS3FFFFF0000(requestBody, laddr, t.Addr); e.Errlog(err) {
		return err
	}
	fmt.Println("发送任务请求包")
	if p, prikey, err := t.rR3FFFFF8000(); e.Errlog(err) {
		return err
	} else {
		fmt.Println("接收任务握手包")
		if err = t.rS3FFFFF4000(p); e.Errlog(err) {
			return err
		} else {
			fmt.Println("回复确认握手包")
			if err = t.rR3FFFFF2000(prikey); e.Errlog(err) {
				return err
			}
			fmt.Println("接收确认确认握手包")
		}
	}
	if err = t.rS3FFFFF1000(); e.Errlog(err) {
		return err
	}

	fmt.Println("握手成功")

	// ---------------接收数据-------------------- //
	var name string
	var fi int64
	var fh *os.File
	var end bool
	for {
		if name, fi, end, err = t.rR3FFFFF0001OR3FFFFFFF00(); e.Errlog(err) {
			return err
		} else {
			if end { //任务结束
				e.Errlog(errors.New("任务结束"))
				return nil
			} else {
				fmt.Println("收到文件信息包", t.Path+`/`+name)
				if fh, err = openFile(t.Path + `/` + name); e.Errlog(err) {
					return err
				}
				// 发送文件开始包
				fmt.Println("发送文件开始包")
				if err = t.rS3FFFFF0002(); e.Errlog(err) {
					return err
				}
			}
		}

		if err = t.rRFileDataPacket(fh, fi); e.Errlog(err) {
			return err
		}

		if err = t.rS3FFFFF00FF(); e.Errlog(err) {
			return err
		}

	}

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
