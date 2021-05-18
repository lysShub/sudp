package sudp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lysShub/e"
	"github.com/lysShub/sudp/internal/packet"
)

func (r *Read) sendSpeedControlPacket(ns int) error {
	var da []byte = []byte{uint8(ns >> 24), uint8(ns >> 16), uint8(ns >> 8), uint8(ns)}

	if da, err = packet.SecureEncrypt(da, r.controlKey); e.Errlog(err) {
		return err
	} else {
		if da, _, _, err = packet.PackagePacket(da, 0x3FFFFF0010, r.key, false); e.Errlog(err) {
			return err
		} else {
			if _, err = r.conn.Write(da); e.Errlog(err) {
				return err
			}
		}
	}
	return nil
}

func (r *Read) sendResendDataPacket(ownRec [][2]int64) error {
	if len(ownRec) == 0 {
		return nil
	}

	var da []byte = make([]byte, 0)
	for _, v := range ownRec {
		da = append(da, uint8((v[0])>>32), uint8((v[0])>>24), uint8((v[0])>>16), uint8((v[0])>>8), uint8((v[0])), uint8((v[1])>>32), uint8((v[1])>>24), uint8((v[1])>>16), uint8((v[1])>>8), uint8((v[1])))
	}

	if da, err = packet.SecureEncrypt(da, r.controlKey); e.Errlog(err) {
		return err
	} else {
		if da, _, _, err = packet.PackagePacket(da, 0x3FFFFF0004, r.key, false); e.Errlog(err) {
			return err
		} else {
			if _, err = r.conn.Write(da); e.Errlog(err) {
				return err
			}
		}
	}

	return nil
}

func (r *Read) sendSchedulPacket(sch int64) error {
	var da []byte
	if da, err = packet.SecureEncrypt([]byte{uint8(sch >> 32), uint8(sch >> 24), uint8(sch >> 16), uint8(sch >> 8), uint8(sch)}, r.controlKey); e.Errlog(err) {
		return err
	} else {
		if da, _, _, err = packet.PackagePacket(da, 0x3FFFFF0008, r.key, false); e.Errlog(err) {
			return err
		} else {
			if _, err = r.conn.Write(da); e.Errlog(err) {
				return err
			}
		}
	}

	return nil
}

func (r *Read) receiverFileInfoOrEndPacket() (string, int64, bool, error) {
	var da []byte
	var l int
	var flag bool = true

	time.AfterFunc(r.TimeOut*2, func() {
		if flag {
			r.conn.Close()
		}
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
		if dl, bias, _, err := packet.ParsePacket(da[:l], r.key); err == nil {

			if bias == 0x3FFFFF0001 { // 文件信息

				if sda, err := packet.SecureDecrypt(da[:dl], r.controlKey); err == nil && len(sda) > 5 {
					flag = false

					go func() { // 回复开始包
						if rda, err := packet.SecureEncrypt(nil, r.controlKey); e.Errlog(err) {
							return
						} else {
							if rda, _, _, err = packet.PackagePacket(rda, 0x3FFFFF0002, r.key, false); e.Errlog(err) {
								return
							} else {
								for i := 0; i < 10; i++ {
									if _, err = r.conn.Write(rda); e.Errlog(err) {
										return
									}
									time.Sleep(time.Millisecond * 10)
								}
							}
						}
					}()

					return string(sda[5:]), int64(sda[0])<<32 + int64(sda[1])<<24 + int64(sda[2])<<16 + int64(sda[3])<<8 + int64(sda[4]), false, nil
				} else {
					e.Errlog(err)
				}

			} else if bias == 0x3FFFFFFF00 { // 任务结束包
				flag = false
				return "", 0, true, nil
			}
		}
	}
}

func (r *Read) sendFileEndPacket() error {
	if da, err := packet.SecureEncrypt(nil, r.controlKey); e.Errlog(err) {
		return err
	} else {
		if da, _, _, err = packet.PackagePacket(da, 0x3FFFFF00FF, r.key, false); e.Errlog(err) {
			return err
		} else {
			for i := 0; i < 20; i++ {
				if _, err = r.conn.Write(da); e.Errlog(err) {
					return err
				}
			}
		}
	}
	return nil
}

func (w *Write) sendFileInfoAndReceiveStartPacket(name string, fs int64) error {
	var sda, rda []byte = []byte{uint8(fs >> 32), uint8(fs >> 24), uint8(fs >> 16), uint8(fs >> 8), uint8(fs)}, nil
	sda = append(sda, []byte(name)...)

	if sda, err = packet.SecureEncrypt(sda, w.controlKey); e.Errlog(err) {
		return err
	} else {
		if sda, _, _, err = packet.PackagePacket(sda, 0x3FFFFF0001, w.key, false); e.Errlog(err) {
			return err
		}
	}

	var flag bool = true
	defer func() { flag = false }()
	go func() {
		for flag {
			if _, err = w.conn.Write(sda); e.Errlog(err) {
				return
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()

	var flag2 bool = true
	time.AfterFunc(w.TimeOut*2, func() {
		if flag2 {
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
			if n, bias, _, err := packet.ParsePacket(rda[:l], w.key); err == nil && bias == 0x3FFFFF0002 {
				if _, err = packet.SecureDecrypt(rda[:n], w.controlKey); err == nil {
					flag2 = false
					return nil
				}
			}
		}
	}
}

// openFile 打开文件, 不存在将会创建、无论声明路径
func openFile(path string) (*os.File, error) {

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(filepath.Dir(path), 0666); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	// 路径已经存在

	fh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}
	return fh, nil
}

// moreDalay time.Sleep()与实际延时有出入, 通常会比设定时间长一定的时间, 本函数返回其多余的时长
// 尤其见于Windows上面, 部分机型可达10ms
func moreDalay() time.Duration {
	var d, total time.Duration = time.Nanosecond, 0
	for i := 0; i < 32; i++ {
		a := time.Now()
		time.Sleep(d)
		total = total + time.Since(a)
	}
	return total >> 5
}
