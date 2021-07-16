// 使用bufio实现
// 弃用
// 使用bufio, 只能从文件的开头开始读取
// 可以尝试使用bytes.Buffer

package file

import (
	"bufio"
	"io"
	"os"

	"github.com/lysShub/sudp/internal/packet"
)

var bufferSize int = 4194304 // 4MB
var n int
var err error

type Rd struct {
	s, e int64 // 缓存中的头尾数据的偏置
	sch  int64
	fs   int64 // 文件大小
	fh   *os.File
	b    *bufio.Reader
}

func NewRead(fh *os.File, bias int64) *Rd {
	var rd = new(Rd)
	rd.fh = fh
	fi, _ := fh.Stat()
	rd.sch = bias

	rd.fs = int64(fi.Size())
	rd.b = bufio.NewReaderSize(fh, bufferSize)
	return rd
}

// 读取
//  返回参数：完整数据包, 数据包中原始数据长度, 是否最后包
//  参数:
// 	d 读取长度为len(d)的数据, 或读取到文件结束(EOF)。确保cap(d) - len(d) > 15
// 	bias 偏置, 读取的第一字节数据的偏置
// 	key 密钥, 为nil则不加密
func (r *Rd) ReadFile(d []byte, bias int64, key []byte) ([]byte, int64, bool, error) {

	if bias < r.s { // 随机读
		return r.randomRead(r.fh, d, bias, key)
	} else {
		r.s = bias

		if n, err = r.b.Read(d); n == len(d) {
			return packet.PackagePacket(d, bias, key, false) // 正常读取

		} else {
			if err == io.EOF { // 恰巧
				return packet.PackagePacket(nil, bias, key, true)

			} else if err == nil && n < len(d) { // 数据读完
				return packet.PackagePacket(d[:n], bias, key, true)

			} else {
				return nil, 0, false, err
			}
		}
	}

}

// 	randomRead 随机读取，适配最后一包
func (r *Rd) randomRead(fh *os.File, d []byte, bias int64, key []byte) ([]byte, int64, bool, error) {

	n, err := fh.ReadAt(d, bias)
	if err != nil {
		if err == io.EOF {

			return packet.PackagePacket(d[:n], bias, key, true)
		}
		return nil, 0, false, err
	}
	return packet.PackagePacket(d, bias, key, false)
}

func WriteFile(d []byte, bias int64, end bool) error {

	return nil
}
