package setting

import (
	"encoding/gob"
	"errors"
	"net"
	"os"
	"runtime"
	"strings"
)

var err error
var fh *os.File
var path string

func init() {
	if runtime.GOOS == "windows" {
		path = os.Getenv("USERPROFILE") + `/sudp.config`

	} else {
		path = `/etc/sudp.config`
	}
}

type S struct {
	MTU       int
	StorePath string
	Auth      []byte
	Crypt     bool
	LIP       net.IP // 本地IP
	LPort     int    // 本地端口
	RPort     int    // 设置默认对端端口
}

func SetMTU(mtu int) error {
	var s *S
	if s, err = Read(); err != nil {
		return err
	}
	s.MTU = mtu
	if err = cover(s); err != nil {
		return err
	}
	return err
}

func SetStorePath(p string) error {
	var s *S
	if s, err = Read(); err != nil {
		return err
	}
	s.StorePath = p
	if err = cover(s); err != nil {
		return err
	}
	return err
}

func SetCrypt(e bool) error {
	var s *S
	if s, err = Read(); err != nil {
		return err
	}
	s.Crypt = e
	if err = cover(s); err != nil {
		return err
	}
	return err
}

func SetAuth(b []byte) error {
	var s *S
	if s, err = Read(); err != nil {
		return err
	}
	s.Auth = b
	if err = cover(s); err != nil {
		return err
	}
	return err
}

func SetLIP(p string) error {
	lip := net.ParseIP(p)
	if lip == nil {
		return errors.New("invalid IP")
	}
	var s *S
	if s, err = Read(); err != nil {
		return err
	}
	s.LIP = lip
	if err = cover(s); err != nil {
		return err
	}
	return err
}

func SetLPort(p int) error {
	var s *S
	if s, err = Read(); err != nil {
		return err
	}
	s.LPort = p
	if err = cover(s); err != nil {
		return err
	}
	return err
}

func SetRPort(p int) error {
	var s *S
	if s, err = Read(); err != nil {
		return err
	}
	s.RPort = p
	if err = cover(s); err != nil {
		return err
	}
	return err
}

// 返回默认参数
func DefaultConfig() *S {
	var s = new(S)
	s.Auth = nil
	s.Crypt = true
	s.MTU = 12000
	s.StorePath = `./`
	s.LIP = nil
	s.LPort = 19986
	s.RPort = 19986
	return s
}

// 读取,
func Read() (*S, error) {

	if fh, err = os.Open(path); err == nil {
		if _, err := fh.Stat(); err != nil {
			return nil, err
		} else {
			dr := gob.NewDecoder(fh)
			var s = new(S)
			if err = dr.Decode(s); err != nil {
				return nil, err
			} else {
				return s, nil
			}
		}
	} else {
		if strings.Contains(err.Error(), "find") {
			cover(DefaultConfig())
			return DefaultConfig(), nil
		} else {
			return nil, err
		}
	}
}

// Init 初始化所有设置
func Init() error {
	return cover(DefaultConfig())
}

// 覆盖写入
func cover(s *S) error {
	if fh, err = os.OpenFile(path, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0666); err != nil {
		return err
	}
	defer fh.Close()
	er := gob.NewEncoder(fh)
	if err = er.Encode(s); err != nil {
		return err
	}
	return nil
}
