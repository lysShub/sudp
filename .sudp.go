package sudp

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/lysShub/e"
)

// type Receive struct {
// 	LanIP net.IP      // 监听的内网IP 不指定网卡可默认
// 	Port  uint16      // 监听的端口 默认19986, 如果设置发送方也应该设置
// 	rconn net.UDPConn // UDPConn, 为unconnected

// }

type SUDP struct {
	Speed int // 实时速度数据, B/s, 更新周期: SCF

	version  uint8         // sudp 版本
	scf      time.Duration // 更新频率，比如实时速度，丢包检查，速度更新的频率等
	conn     *net.UDPConn  // 发送\接收端的UDPconn
	key      []byte        // 密钥， 为nil表示不加密
	mtu      int           // mtu, 单位字节
	pubkey   []byte        //
	prikey   []byte
	schedule int64 // 进度

	// storepath string // 接收端储存路径(文件夹、必须已存在)

	//
	//
	// Receive
}

// Tasker 表示一个传输任务
//
type Tasker struct {
	Speed    int          // 实时速度，B/s
	Schedule int64        // 进度
	raddr    *net.UDPAddr // 对方地址
	key      []byte       // 密钥
	prikey   []byte       // 私钥
	pubkey   []byte       // 公钥

}

var err error

// Send 发送
//  key为nil表示不加密
func (s *SUDP) Send(path, name string, conn *net.UDPConn, key []byte) (int64, error) {

	s.Speed = 1024 // 1MB/s 初始速度
	s.conn = conn
	s.scf = time.Duration(time.Second) // 先默认
	s.version = 0b0000000
	s.mtu = 1372
	if key != nil {
		if len(key) != 16 {
			return 0, errors.New("invlid secret key!")
		}
		s.key = key[:16]
	}

	/* ------------------------开始--------------------- */
	if err = s.sHandshakePacket(time.Hour, func(requestBody []byte) bool { return true }); err != nil {
		return 0, err
	}
	fmt.Println("收到确认握手包")

	fi, err := os.Stat(path + `/` + name)
	if e.Errlog(err) {
		return 0, err
	}
	if err = s.sFileInfoPacket(path, name, fi.Size()); e.Errlog(err) {
		return 0, err
	}

	fmt.Println("握手成功")
	return 0, nil

	var fh *os.File
	if fh, err = os.Open(path + `/` + name); err != nil {
		return 0, err
	}
	var n int64
	if n, err = s.sFileDataPacket(fh, fi.Size()); err != nil {
		return 0, err
	}
	return n, nil
}

// Receive 接收数据 当一个文件传输完成(或中止)即可退出
//  basePath 是文件储存路径
func (s *SUDP) Receive(basePath string, conn *net.UDPConn) error {
	s.scf = time.Duration(time.Second) // 先默认
	s.conn = conn
	/* --------------------------开始----------------------------------- */

	if err = s.rHandshakePacket(nil); err != nil {
		return err
	}
	fmt.Println("发送确认握手包")
	var path string
	var fs int64
	// var fh *os.File

	if path, fs, err = s.rFileInfoPacket(); e.Errlog(err) {
		return err
	}
	fmt.Println("收到文件信息包")

	fmt.Println(string(path), fs)
	fmt.Println("握手成功，退出")
	return nil

	if _, err = openFile(basePath + `/` + path); e.Errlog(err) {
		return err
	}

	if err = s.rStartPacket(); e.Errlog(err) {
		return err
	}

	return nil
}
