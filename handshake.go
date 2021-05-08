package sudp

/* 协议握手 */

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sudp/internal/crypter"
	"sudp/internal/packet"
	"time"

	"github.com/lysShub/e"
)

// SendHandshake 握手
func (s *SUDP) SendHandshake(laddr, raddr *net.UDPAddr, requestBody []byte) error {
	var rda, sda []byte = make([]byte, 1500), make([]byte, 0, 64)
	if s.conn, err = net.DialUDP("udp", laddr, raddr); e.Errlog(err) {
		return err
	}
	var flag bool = true
	defer func() { flag = false }()
	var n int

	// 回复
	var ch chan error = make(chan error, 1)
	go func() {
		for flag {
			if _, err = s.conn.Write(sda); err != nil {
				ch <- err
				return
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()

	// 请求
	if sda, _, _, err = packet.PackageDataPacket(requestBody, 0x3FFFFF0000, nil, false); err != nil {
		return err
	}

	// 握手
	var priKey []byte
	var step int = 0
	time.AfterFunc(s.TimeOut, func() {
		if step < 1 {
			fmt.Println("关闭conn")
			s.conn.Close()
		}
	})
	for {
		if len(ch) != 0 {
			return <-ch
		}
		rda = make([]byte, 1500)
		if n, err = s.conn.Read(rda); e.Errlog(err) {
			return err
		}
		if _, bias, _, err := packet.ParseDataPacket(rda[:n], nil); e.Errlog(err) {
			return err
		} else if bias == 0x3FFFFF8000 {
			step = 1
			if rda[0] != Version { // 版本不相同
				sda = []byte{10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 252, 0, 8, 106, 249, 147, 14}
				time.Sleep(time.Millisecond * 50)
				return errors.New("incompatible protocol version")
			} else {
				mtu := int(rda[1])<<8 + int(rda[2])
				if mtu < s.MTU {
					s.MTU = mtu
				}
				if rda[3] != 0 { // 加密
					var pubkey []byte
					if priKey, pubkey, err = crypter.RsaGenKey(); e.Errlog(err) {
						return err
					} else {
						if sda, _, _, err = packet.PackageDataPacket(append([]byte{0, uint8(s.MTU >> 8), uint8(s.MTU)}, pubkey...), 0x3FFFFF4000, nil, false); e.Errlog(err) {
							return err
						}
					}
				} else {
					if sda, _, _, err = packet.PackageDataPacket(append([]byte{0, uint8(s.MTU >> 8), uint8(s.MTU)}, make([]byte, 162)...), 0x3FFFFF4000, nil, false); e.Errlog(err) {
						return err
					}
				}
			}
			break
		}
	}

	// 确认确认握手
	time.AfterFunc(s.TimeOut, func() {
		if step < 2 {
			s.conn.Close()
		}
	})
	for {
		if len(ch) != 0 {
			return <-ch
		}
		rda = make([]byte, 1500)
		if n, err = s.conn.Read(rda); err != nil {
			if strings.Contains(err.Error(), "closed") {
				return errors.New("timeout")
			} else if e.Errlog(err) {
				return err
			}
		}

		if dl, bias, _, err := packet.ParseDataPacket(rda[:n], nil); e.Errlog(err) {
			return err
		} else if bias == 0x3FFFFF2000 {
			step = 2
			if priKey != nil {
				if rkey, err := crypter.RsaDecrypt(rda[:dl], priKey); e.Errlog(err) {
					return err
				} else {
					s.key = rkey
				}
			} else {
				s.key = nil
			}
			break
		} else {
			fmt.Println("确认确认握手收到", bias)
		}
	}

	// 开始
	if sda, _, _, err = packet.PackageDataPacket(nil, 0x3FFFFF1000, nil, false); e.Errlog(err) {
		return err
	}
	time.Sleep(time.Millisecond * 60)

	return nil
}

// ReceiveHandshake 接收握手
func (s *SUDP) ReceiveHandshake(laddr *net.UDPAddr, f func(requestBody []byte) bool) error {
	var rda, sda []byte = make([]byte, 1500), nil
	var n int
	var flag bool = true
	defer func() { flag = false }()

	if s.conn, err = net.ListenUDP("udp", laddr); e.Errlog(err) {
		return err
	}

	var raddr *net.UDPAddr
	for {
		rda = make([]byte, 1500)
		if n, raddr, err = s.conn.ReadFromUDP(rda); e.Errlog(err) {
			return err
		}
		if dl, bias, _, err := packet.ParseDataPacket(rda[:n], nil); err == nil {
			if bias == 0x3FFFFF0000 {
				if f(rda[:dl]) {
					s.conn.Close()
					break // 接受
				} else {
					e.Errlog(errors.New("Authentication failed, raddr:" + raddr.String()))
				}
			}
		}
	}
	if s.conn, err = net.DialUDP("udp", laddr, raddr); e.Errlog(err) { // 替换为Connected UDP
		return err
	}

	// 握手
	var isEncrypto uint8 = 0x0
	if s.Encrypt {
		isEncrypto = 0xf
		s.key = s.createKey()
	}
	if sda, _, _, err = packet.PackageDataPacket([]byte{Version, uint8(s.MTU >> 8), uint8(s.MTU), isEncrypto}, 0x3FFFFF8000, nil, false); e.Errlog(err) {
		return err
	}

	/* 回复 */
	var ch chan error = make(chan error, 1)
	go func() {
		for flag {
			if _, err = s.conn.Write(sda); e.Errlog(err) {
				ch <- err
				return
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()

	// 确认握手
	var step int = 0
	time.AfterFunc(s.TimeOut, func() {
		if step < 1 {
			s.conn.Close()
		}
	})
	for {
		if len(ch) != 0 {
			return <-ch
		}
		rda = make([]byte, 1500)
		if n, err = s.conn.Read(rda); err != nil {
			if strings.Contains(err.Error(), "closed") {
				return errors.New("timeout")
			} else if e.Errlog(err) {
				return err
			}
		}

		if dl, bias, _, err := packet.ParseDataPacket(rda[:n], nil); e.Errlog(err) {
			return err

		} else if bias == 0x3FFFFF4000 {
			if rda[0] != 0 { // 握手代码
				return errors.New("HandshakeCode: " + strconv.Itoa(int(rda[0])))
			}
			s.MTU = int(rda[1])<<8 + int(rda[2])
			var tda []byte = make([]byte, 128)
			if s.key != nil {
				if tda, err = crypter.RsaEncrypt(s.key, rda[3:dl]); e.Errlog(err) {
					return err
				}
			}
			// 确认确认握手
			if sda, _, _, err = packet.PackageDataPacket(tda, 0x3FFFFF2000, nil, false); e.Errlog(err) {
				return err
			}
			step = 1
			break
		}
	}

	// 任务开始包
	time.AfterFunc(s.TimeOut, func() {
		if step < 2 {
			s.conn.Close()
		}
	})
	for {
		if len(ch) != 0 {
			return <-ch
		}
		rda = make([]byte, 1500)
		if n, err = s.conn.Read(rda); err != nil {
			if strings.Contains(err.Error(), "closed") {
				return errors.New("timeout")
			} else if e.Errlog(err) {
				return err
			}
		}
		if _, bias, _, err := packet.ParseDataPacket(rda[:n], nil); e.Errlog(err) {
			return err
		} else if bias == 0x3FFFFF1000 {
			step = 2
			break // 握手完成
		}
	}

	return nil
}

func (s *SUDP) createKey() []byte {
	_, b, err := crypter.RsaGenKey()
	var t [16]byte
	if err != nil {
		t = md5.Sum([]byte(time.Now().String()))
	} else {
		t = md5.Sum(b)
	}
	return t[:]
}
