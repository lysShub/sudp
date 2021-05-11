package file

import (
	"fmt"
	"io"
	"os"

	"github.com/lysShub/sudp/internal/com"
	"github.com/lysShub/sudp/internal/packet"
)

var err error

// 此包用于文件的读写
// 使用缓存，使得4K读写性能达到顺序读写的性能
// 适用于大部分为顺序读写的情况
//                 慢速                快速
//只读			耗时 58 速度 131 || 耗时 7 速度 1091
//读写			耗时 192 速度 39 || 耗时 58 速度 131
//读写（加密）	 耗时 261 速度 29 || 耗时 70 速度 109

// Rd  Read 文件读取
type Rd struct {
	Fh *os.File

	fs               int64    // 文件大小
	initflag         bool     // 初始化标志
	fm               bool     // 大文件缓存读取模式，文件大于48MB自动开启
	bs               int64    // 快速读取模式下的暂存数据块大小
	block            []byte   // 缓存数据块
	rang             [2]int64 // 记录block中数据的位置
	smallProbability bool
}

// init 初始化函数
func (f *Rd) init() error {
	if !f.initflag {
		fmt.Println("启动")

		fi, err := f.Fh.Stat()
		if err != nil {
			return err
		}
		f.fs = int64(fi.Size())
		if (f.fs >> 24) >= 0b11 {
			f.fm = true
			f.bs = 4194304               //4194304 4MB
			f.block = make([]byte, f.bs) //
			if f.fs%f.bs == 0 {
				f.smallProbability = true
			}
		} else {
			f.fm = false
		}
		f.initflag = true
	}
	return nil
}

// ReadFile 读取文件；
//   返回：打包好数据包，原始数据长度，是否最后包。
//   参数d应该有足够的容量(len+15); 否则会浪费内存。正常情况下, 最后一个数据包读取的数据长度可能和len(d)不相同
func (f *Rd) ReadFile(d []byte, bias int64, key []byte) ([]byte, int64, bool, error) {
	if err = f.init(); err != nil {
		return nil, 0, false, err
	}

	// 启用快速读取模式
	if f.fm {

		if bias < f.rang[0] {
			return f.randomRead(f.Fh, d, bias, key)
		}

		l := int64(len(d))
		if f.rang[1] < bias+l-1 {

			_, err := f.Fh.ReadAt(f.block, bias)
			if err != nil {
				if err == io.EOF { // 剩余文件不足以读取16MB的数据块
					return f.randomRead(f.Fh, d, bias, key)
				} else if com.Errorlog(err) {
					return nil, 0, false, err
				}
			}
			f.rang[0], f.rang[1] = bias, bias+f.bs-1 // 更新记录

		}
		copy(d, f.block[bias-f.rang[0]:])

		// 16MB数据块恰好读完文件数据，且此数据包恰好读完数据块中最后数据
		if f.smallProbability && bias+l+1 == f.fs {
			return packet.PackagePacket(d, bias, key, true)
		}
		return packet.PackagePacket(d, bias, key, false)
	}

	// 不启用快速读取模式
	return f.randomRead(f.Fh, d, bias, key)
}

// readfile
// 	随机读取，适配最后一包
func (f *Rd) randomRead(fh *os.File, d []byte, bias int64, key []byte) ([]byte, int64, bool, error) {

	_, err := fh.ReadAt(d, bias)
	if err != nil {
		if err == io.EOF {
			if f.fs-bias == 1 {
				d = nil
				return nil, 0, true, nil
			}
			d = make([]byte, f.fs-bias, f.fs-bias+9)
			_, err = fh.ReadAt(d, bias)
			if err != nil {
				return nil, 0, false, err
			}
			return packet.PackagePacket(d, bias, key, true)

		}
		return nil, 0, false, err
	}
	return packet.PackagePacket(d, bias, key, false)
}

/* ---------------------------------------------------------------------------------------------- */

// Wt Write 文件写入，传入标准数据包即可
// 传入的正常数据一定要写
type Wt struct {
	// 文件句柄
	Fh *os.File

	initflag bool     // 初始化标志
	bs       int64    // block size 快速写入模式下的暂存数据块大小
	block    []byte   // 储存数据块，存入暂存数据必须连续且小于
	rang     [2]int64 // 记录block中数据的位置
	rbias    int64    // 记录block中有效数据长度
}

// init 初始化函数
func (f *Wt) init() {
	if !f.initflag {
		fmt.Println("启动")

		f.bs = 4194304
		f.block = make([]byte, f.bs)
		f.rang = [2]int64{
			0, f.bs,
		}
		f.initflag = true
	}
}

// WriteFile 写入文件
//  传入参数: 原始数据, 偏置, 是否清空缓存
func (f *Wt) WriteFile(d []byte, bias int64, end bool) error {
	f.init()

	dl := int64(len(d))

	//重置缓存块
	if f.rang[1] < bias+dl-1 || end {
		_, err = f.Fh.WriteAt(f.block[:f.rbias], f.rang[0])

		// 重置
		f.rang[0] = bias
		f.rang[1] = bias + f.bs
		copy(f.block[0:dl], d)
		f.rbias = dl
		if end { // 清空缓存
			_, err = f.Fh.WriteAt(f.block[:f.rbias], f.rang[0])
		}

	} else {
		if bias >= f.rang[0] { //存入缓存块
			copy(f.block[bias-f.rang[0]:bias-f.rang[0]+dl], d)
			f.rbias = f.rbias + dl
		} else { // 非缓存范围 直接写入
			_, err = f.Fh.WriteAt(d, bias)
		}
	}
	return err
}
