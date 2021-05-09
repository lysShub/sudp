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

// sendHandshake 握手, 接收方
func (r *Read) sendHandshake(requestBody []byte) error {
	var rda, sda []byte = make([]byte, 1500), make([]byte, 0, 64)
	if r.conn, err = net.DialUDP("udp", r.Laddr, r.Raddr); e.Errlog(err) {
		return err
	}
	var flag bool = true
	defer func() { flag = false }()
	var n int

	// 请求
	if sda, _, _, err = packet.PackageDataPacket(requestBody, 0x3FFFFF0000, nil, false); err != nil {
		return err
	}

	// 回复
	var ch chan error = make(chan error, 1)
	go func() {
		for flag {
			if _, err = r.conn.Write(sda); err != nil {
				ch <- err
				return
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()

	// 握手
	var priKey []byte
	var encryp bool = false // 文件数据是否加密
	var step int = 0
	time.AfterFunc(r.TimeOut, func() {
		if step < 1 {
			fmt.Println("关闭conn")
			r.conn.Close()
		}
	})
	for {
		if len(ch) != 0 {
			return <-ch
		}
		rda = make([]byte, 1500)
		if n, err = r.conn.Read(rda); e.Errlog(err) {
			return err
		}
		if _, bias, _, err := packet.ParseDataPacket(rda[:n], nil); e.Errlog(err) {
			return err
		} else if bias == 0x3FFFFF8000 {
			step = 1
			if rda[0] != Version { // 版本不相同
				sda = []byte{10, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 255, 255, 252, 0, 8, 106, 249, 147, 14}
				r.conn.Write(sda)
				time.Sleep(time.Millisecond * 50) // 重复发送确保抵达
				return errors.New("incompatible protocol version")
			} else {
				mtu := int(rda[1])<<8 + int(rda[2])
				if mtu < r.MTU {
					r.MTU = mtu
				}
				if rda[3] != 0 { // 文件数据加密
					encryp = true
				}
				var pubkey []byte
				if priKey, pubkey, err = crypter.RsaGenKey(); e.Errlog(err) {
					return err
				} else {
					if sda, _, _, err = packet.PackageDataPacket(append([]byte{0, uint8(r.MTU >> 8), uint8(r.MTU)}, pubkey...), 0x3FFFFF4000, nil, false); e.Errlog(err) {
						return err
					}
					r.conn.Write(sda)
				}
			}
			break
		}
	}

	// 确认确认握手
	time.AfterFunc(r.TimeOut, func() {
		if step < 2 {
			r.conn.Close()
		}
	})
	for {
		if len(ch) != 0 {
			return <-ch
		}
		rda = make([]byte, 1500)
		if n, err = r.conn.Read(rda); err != nil {
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
			if rkey, err := crypter.RsaDecrypt(rda[:dl], priKey); e.Errlog(err) {
				return err
			} else {
				r.controlKey = rkey
				fmt.Println("密钥", r.key)
			}
			if encryp {
				r.key = r.controlKey
			} else {
				r.key = nil
			}
			break
		}
	}

	// 开始
	if sda, _, _, err = packet.PackageDataPacket(nil, 0x3FFFFF1000, r.key, false); e.Errlog(err) {
		return err
	}
	fmt.Println("开始报长度", len(sda))
	for i := 0; i < 5; i++ {
		r.conn.Write(sda)
	}
	return nil
}

// receiveHandshake 接收握手, 发送方
func (w *Write) receiveHandshake(f func(requestBody []byte) bool) error {
	var rda, sda []byte = make([]byte, 1500), nil
	var n int
	var flag bool = true
	defer func() { flag = false }()

	if w.conn, err = net.ListenUDP("udp", w.Laddr); e.Errlog(err) {
		return err
	}

	var raddr *net.UDPAddr
	for {
		rda = make([]byte, 1500)
		if n, raddr, err = w.conn.ReadFromUDP(rda); e.Errlog(err) {
			return err
		}
		if dl, bias, _, err := packet.ParseDataPacket(rda[:n], nil); err == nil {
			if bias == 0x3FFFFF0000 {
				if f(rda[:dl]) {
					w.conn.Close()
					break // 接受
				} else {
					e.Errlog(errors.New("Authentication failed, raddr:" + raddr.String()))
				}
			}
		}
	}
	w.Raddr = raddr
	if w.conn, err = net.DialUDP("udp", w.Laddr, raddr); e.Errlog(err) { // 替换为Connected UDP
		return err
	}

	// 握手
	w.controlKey = createKey()
	fmt.Println("密钥", w.key)
	var isEncrypto uint8 = 0x0
	if w.Encrypt {
		isEncrypto = 0xf
		w.key = w.controlKey
	}
	if sda, _, _, err = packet.PackageDataPacket([]byte{Version, uint8(w.MTU >> 8), uint8(w.MTU), isEncrypto}, 0x3FFFFF8000, nil, false); e.Errlog(err) {
		return err
	}

	/* 回复 */
	var ch chan error = make(chan error, 1)
	go func() {
		for flag {
			if _, err = w.conn.Write(sda); e.Errlog(err) {
				ch <- err
				return
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()

	// 确认握手
	var step int = 0
	time.AfterFunc(w.TimeOut, func() {
		if step < 1 {
			w.conn.Close()
		}
	})
	for {
		if len(ch) != 0 {
			return <-ch
		}
		rda = make([]byte, 1500)
		if n, err = w.conn.Read(rda); err != nil {
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
			w.MTU = int(rda[1])<<8 + int(rda[2])

			var tda []byte = make([]byte, 128)
			if tda, err = crypter.RsaEncrypt(w.key, rda[3:dl]); e.Errlog(err) {
				return err
			}
			// 确认确认握手
			if sda, _, _, err = packet.PackageDataPacket(tda, 0x3FFFFF2000, nil, false); e.Errlog(err) {
				return err
			}
			w.conn.Write(sda)
			step = 1
			break
		}
	}

	// 任务开始包
	time.AfterFunc(w.TimeOut, func() {
		if step < 2 {
			w.conn.Close()
		}
	})
	for {
		if len(ch) != 0 {
			return <-ch
		}
		rda = make([]byte, 1500)
		if n, err = w.conn.Read(rda); err != nil {
			if strings.Contains(err.Error(), "closed") {
				return errors.New("timeout")
			} else if e.Errlog(err) {
				return err
			}
		}
		if _, bias, _, err := packet.ParseDataPacket(rda[:n], w.key); e.Errlog(err) {
			return err
		} else if bias == 0x3FFFFF1000 {
			step = 2
			break // 收到开始包, 握手完成
		}
	}

	return nil
}

func createKey() []byte {
	_, b, err := crypter.RsaGenKey()
	var t [16]byte
	if err != nil {
		t = md5.Sum([]byte(time.Now().String()))
	} else {
		t = md5.Sum(b)
	}
	return t[:]
}
