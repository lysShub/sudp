package tasker

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sudp"
	"sudp/internal/crypter"
	"sudp/internal/file"
	"sudp/internal/packet"
	"time"

	"github.com/lysShub/e"
)

type Sender struct {
	/* ----------------------------- */
	Speed    int   // 实时速度，B/s
	Schedule int64 // 进度

	/* ----------------------------- */
	key          []byte        // 密钥
	conn         *net.UDPConn  // 发送方初始为unconnected UDPConn
	matchTimeOut time.Duration // 匹配超时, 一般设置较长时间
	replyTimeOut time.Duration // 回复超时时间, 较短时间
	mtu          int           // 链路MTU
	ts           time.Duration // 速度控制
}

func (s *Sender) sinit() {
	s.matchTimeOut = time.Minute * 15
	s.replyTimeOut = time.Second
	s.mtu = 1372
	s.Speed = 131072 // 0.13MB/s
}

// sR3FFFFF0000 接收请求包
//  返回connected的UDPConn
//  会一直阻塞知道收到正确的请求, 权鉴不成功忽略
func (s *Sender) sR3FFFFF0000(laddr *net.UDPAddr, f func(requestBody []byte) bool) error {

	// 使用默认网卡
	if s.conn, err = net.ListenUDP("udp", laddr); e.Errlog(err) {
		return err
	}
	var flag bool = true
	var ch chan error = make(chan error, 1)
	var n int
	var da []byte
	var raddr *net.UDPAddr

	go func() {
		for flag {
			da = make([]byte, 1500)
			if err = s.conn.SetReadDeadline(time.Now().Add(s.matchTimeOut)); e.Errlog(err) {
				ch <- err
				return
			}
			if n, raddr, err = s.conn.ReadFromUDP(da); err != nil {
				if strings.Contains(err.Error(), "timeout") { // read udp [::]:19986: i/o timeout
					continue
				} else if e.Errlog(err) {
					ch <- err
					return
				}
			}
			if dl, bias, _, err := packet.ParseDataPacket(da[:n], nil); err == nil {
				if bias == 0x3FFFFF0000 {
					if f(da[:dl]) {
						ch <- nil
						return
					} else {
						e.Errlog(errors.New("Authentication failed, raddr:" + raddr.String()))
					}
				}
			}

		}
	}()

	select {
	case err = <-ch:
		if err == nil {
			s.conn.Close()
			if s.conn, err = net.DialUDP("udp", laddr, raddr); e.Errlog(err) {
				return err
			} else {
				return nil
			}
		}
		return err
	case <-time.After(s.matchTimeOut):
		flag = false
		return errors.New("timeout")
	}
}

// sS3FFFFF8000s 发送握手包
func (s *Sender) sS3FFFFF8000(encrypt bool) error {

	var da []byte

	var isEncrypto uint8 = 0x0
	if encrypt {
		isEncrypto = 0xf
		tem := s.createKey() // 生成密钥key
		s.key = tem[:]
	}

	if da, _, _, err = packet.PackageDataPacket([]byte{sudp.Version, uint8(s.mtu >> 8), uint8(s.mtu), isEncrypto}, 0x3FFFFF8000, nil, false); e.Errlog(err) {
		return err
	}
	for i := 0; i < reliable; i++ {
		if _, err = s.conn.Write(da); e.Errlog(err) {
			return err
		}
	}
	return nil
}

// sR3FFFFF4000 接收确认握手包
//  返回公钥(不加密为nil)
func (s *Sender) sR3FFFFF4000() ([]byte, error) {

	var flag bool = true
	var ch chan error = make(chan error, 1)
	var n int
	var da []byte
	var pubkey []byte
	go func() {
		for flag {
			da = make([]byte, 1500)
			if err = s.conn.SetReadDeadline(time.Now().Add(s.replyTimeOut)); e.Errlog(err) {
				ch <- err
				return
			}
			if n, err = s.conn.Read(da); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					ch <- errors.New("timeout")
					return
				} else if e.Errlog(err) {
					ch <- err
					return
				}

			} else {

				if dl, bias, _, err := packet.ParseDataPacket(da[:n], nil); err == nil {
					if bias == 0x3FFFFF4000 {
						if da[0] != 0 { // 握手代码
							ch <- errors.New("HandshakeCode: " + strconv.Itoa(int(da[0])))
							return
						}
						s.mtu = int(da[1])<<8 + int(da[2])
						if s.key != nil {
							pubkey = da[3:dl]
						}
						ch <- nil
						return
					}
				}
			}
		}
	}()
	select {
	case err = <-ch:
		if err != nil {
			return nil, err
		}
		return pubkey, err
	case <-time.After(s.replyTimeOut * 2):
		flag = false
		return nil, errors.New("timeout")
	}
}

// sS3FFFFF2000 发送确认确认握手包
func (s *Sender) sS3FFFFF2000(publicKey []byte) error {
	var da []byte = make([]byte, 128)
	if s.key != nil {
		if da, err = crypter.RsaEncrypt(s.key, publicKey); e.Errlog(err) {
			return err
		}
	}

	if da, _, _, err = packet.PackageDataPacket(da, 0x3FFFFF2000, nil, false); e.Errlog(err) {
		return err
	}
	for i := 0; i < reliable; i++ {
		if _, err = s.conn.Write(da); e.Errlog(err) {
			return err
		}
	}
	return nil
}

// sR3FFFFF1000 接收任务开始包
func (s *Sender) sR3FFFFF1000() error {
	var flag bool = true
	var ch chan error = make(chan error, 1)
	var n int
	var da []byte
	go func() {
		for flag {
			da = make([]byte, 1500)
			if err = s.conn.SetReadDeadline(time.Now().Add(s.replyTimeOut)); e.Errlog(err) {
				ch <- err
				return
			}
			if n, err = s.conn.Read(da); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					ch <- errors.New("timeout")
					return
				} else if e.Errlog(err) {
					ch <- err
					return
				}
			} else {
				if _, bias, _, err := packet.ParseDataPacket(da[:n], nil); err == nil {
					if bias == 0x3FFFFF1000 {
						ch <- nil
						return
					}
				}
			}
		}
	}()
	select {
	case err = <-ch:
		return err
	case <-time.After(s.replyTimeOut * 2):
		flag = false
		return errors.New("timeout")
	}
}

// sS3FFFFFFF00 发送任务结束包
func (s *Sender) sS3FFFFFFF00() error {
	var da []byte
	if da, _, _, err = packet.PackageDataPacket(nil, 0x3FFFFFFF00, nil, false); e.Errlog(err) {
		return err
	}
	for i := 0; i < reliable; i++ {
		if _, err = s.conn.Write(da); e.Errlog(err) {
			return err
		}
	}
	return nil
}

/* -------------------------------------------------- */

// sS3FFFFF0001 发送文件信息包
func (s *Sender) sS3FFFFF0001(name string, fs int64) error {

	var da []byte = []byte{
		uint8(fs >> 32), uint8(fs >> 24), uint8(fs >> 16), uint8(fs >> 8), uint8(fs),
	}
	da = append(da, []byte(name)...)
	if da, _, _, err = packet.PackageDataPacket(da, 0x3FFFFF0001, s.key, false); e.Errlog(err) {
		return err
	} else {
		for i := 0; i < reliable; i++ {
			if _, err = s.conn.Write(da); e.Errlog(err) {
				return err
			}
		}
	}
	return nil
}

// sR3FFFFF0008 接收文件开始包
func (s *Sender) sR3FFFFF0002() error {
	var flag bool = true
	var ch chan error = make(chan error, 1)
	go func() {
		var da []byte
		var n int
		for flag {
			da = make([]byte, 1500)
			if err = s.conn.SetReadDeadline(time.Now().Add(s.replyTimeOut)); e.Errlog(err) {
				ch <- err
				return
			}
			if n, err = s.conn.Read(da); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					ch <- errors.New("timeout")
					return
				} else if e.Errlog(err) {
					ch <- err
					return
				}
			} else {
				if _, bias, _, err := packet.ParseDataPacket(da[:n], s.key); err != nil {
					e.Errlog(err)
					continue
				} else if bias == 0x3FFFFF0002 {
					e.Errlog(errors.New("收到开始包"))
					ch <- nil
				}
			}

		}
	}()

	select {
	case err = <-ch:
		return err
	case <-time.After(s.replyTimeOut * 2):
		flag = false
		return errors.New("timeout")
	}
}

// sFileDataPacket 发送文件数据包
//  返回值为0表示发生错误，非0表示传输结束、小于fs表示传输中止、等于fs表示传输完成
func (s *Sender) sSFileDataPacket(fh *os.File, fs int64) (int64, error) {
	r := new(file.Rd)
	r.Fh = fh

	var ch chan error = make(chan error, 2)
	var reSend chan []byte = make(chan []byte, 64)
	var end chan int64 = make(chan int64, 1)
	var flag bool = true

	// 接收
	go func() {
		var da []byte
		var bias, dl int64
		var l int

		for flag {

			da = make([]byte, 1500)
			if err = s.conn.SetReadDeadline(time.Now().Local().Add(s.replyTimeOut * 15)); e.Errlog(err) {
				ch <- err
				return
			}
			if l, err = s.conn.Read(da); err != nil {
				if strings.Contains(err.Error(), "timeout") {
					// 15s起码没收到进度包，中止传输
					// ch <- errors.New("transmission aborted")
					ch <- nil
					e.Errlog(errors.New("起码15s没有收到进度包"))
					return
				} else {
					ch <- err
					return
				}
			} else {

				if dl, bias, _, err = packet.ParseDataPacket(da[:l], s.key); err == nil {

					if bias == 0x3FFFFF0004 { // 文件重发包
						fmt.Println("接收到文件重发包")
						reSend <- da[:dl]

					} else if bias == 0x3FFFFF0008 { // 文件进度包
						s.Schedule = int64(da[0])<<32 + int64(da[1])<<24 + int64(da[2])<<16 + int64(da[3])<<8 + int64(da[4])
						fmt.Println("接收到文件进度包", s.Schedule)
					} else if bias == 0x3FFFFF00FF { // 文件结束包
						fmt.Println("收到文件结束包")
						end <- fs
						return
					} else if bias == 0x3FFFFF0010 {
						fmt.Println("收到速度控制包", s.Speed, int(da[0])<<24+int(da[1])<<16+int(da[2])<<8+int(da[3]))

						s.Speed = int(da[0])<<24 + int(da[1])<<16 + int(da[2])<<8 + int(da[3])

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
		s.ts = time.Duration(1e9 * s.mtu / s.Speed) // 控制传输速度
		go func() {
			for flag {
				s.ts = time.Duration(1e9 * s.mtu / s.Speed)
				time.Sleep(time.Millisecond * 100) // 速度变换精度，通常小于几倍策略中的统计周期
			}
		}()
		go func() {
			for flag {
				time.Sleep(time.Second * 2)
			}
		}()

		for bias = int64(0); bias < fs; {
			d = make([]byte, s.mtu, s.mtu+25)

			if len(reSend) != 0 {
				for i := 0; i < len(reSend); i++ {
					if err = s.sS3FFFFF0004(<-reSend, r); e.Errlog(err) {
						continue
					}
				}
			}

			if d, dl, sEnd, err = r.ReadFile(d, bias, s.key); e.Errlog(err) {
				ch <- err
				return
			}
			if _, err = s.conn.Write(d); e.Errlog(err) {
				ch <- err
				return
			}
			if sEnd { // 主进程发送结束
				break
			}
			bias = bias + dl
			time.Sleep(s.ts) // 速度控制
			if sEnd {
				// 由于收到之后的数据包后才会重发之前缺失的数据，如果最后包没有到达，则会导致传输挂起
				for {
					fmt.Println("------------------------------------发送完成-------------------------------")

					time.Sleep(time.Millisecond * 500)
					if _, err = s.conn.Write(d); e.Errlog(err) {
						ch <- err
						return
					}
				}
			}
		}
	}()

	select {
	case err = <-ch:
		flag = false
		return s.Schedule, err
	case r := <-end:
		flag = false
		return r, nil
	}
}

// sS3FFFFF0004 发送重发数据包
func (s *Sender) sS3FFFFF0004(da []byte, r *file.Rd) error {

	var sb, eb int64
	var d []byte
	for i := 9; i <= len(da); i = i + 10 {

		sb = int64(da[i-9])<<32 + int64(da[i-8])<<24 + int64(da[i-7])<<16 + int64(da[i-6])<<8 + int64(da[i-5])
		eb = int64(da[i-4])<<32 + int64(da[i-3])<<24 + int64(da[i-2])<<16 + int64(da[i-1])<<8 + int64(da[i-0])

		for i := sb; i <= eb; i = i + int64(s.mtu) {
			if int64(s.mtu)+i-1 > eb {
				d = make([]byte, eb-i+1)
			} else {
				d = make([]byte, s.mtu)
			}
			if d, _, _, err = r.ReadFile(d, i, s.key); e.Errlog(err) {
				return err
			}
			if _, err = s.conn.Write(d); e.Errlog(err) {
				return err
			}
			time.Sleep(s.ts)
		}
	}

	return nil
}

func (s *Sender) createKey() [16]byte {
	_, b, err := crypter.RsaGenKey()
	if err != nil {
		return md5.Sum([]byte(time.Now().String()))
	}
	return md5.Sum(b)
}
