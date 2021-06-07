package file

import (
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
func (r *Rd) init() error {
	if !r.initflag {

		fi, err := r.Fh.Stat()
		if err != nil {
			return err
		}
		r.fs = int64(fi.Size())
		if (r.fs >> 24) >= 0b11 {
			r.fm = true
			r.bs = 4194304               //4194304 4MB
			r.block = make([]byte, r.bs) //
			if r.fs%r.bs == 0 {
				r.smallProbability = true
			}
		} else {
			r.fm = false
		}
		r.initflag = true
	}
	return nil
}

// ReadFile 读取文件；
//   返回：打包好数据包，原始数据长度，是否最后包。
//   参数d应该有足够的容量(len+15); 否则会浪费内存。正常情况下, 最后一个数据包读取的数据长度可能和len(d)不相同
func (r *Rd) ReadFile(d []byte, bias int64, key []byte) ([]byte, int64, bool, error) {
	if err = r.init(); err != nil {
		return nil, 0, false, err
	}

	return r.randomRead(r.Fh, d, bias, key)

	// 启用快速读取模式
	if r.fm {

		if bias < r.rang[0] {
			return r.randomRead(r.Fh, d, bias, key)
		}

		l := int64(len(d))
		if r.rang[1] < bias+l-1 { // 读取到缓存块

			_, err := r.Fh.ReadAt(r.block, bias)
			if err != nil {
				if err == io.EOF { // 剩余文件不足以读取16MB的数据块
					return r.randomRead(r.Fh, d, bias, key)
				} else if com.Errorlog(err) {
					return nil, 0, false, err
				}
			}
			r.rang[0], r.rang[1] = bias, bias+r.bs-1 // 更新记录

		}
		copy(d, r.block[bias-r.rang[0]:])

		// 16MB数据块恰好读完文件数据，且此数据包恰好读完数据块中最后数据
		if r.smallProbability && bias+l+1 == r.fs {
			return packet.PackagePacket(d, bias, key, true)
		}
		return packet.PackagePacket(d, bias, key, false)
	}

	// 不启用快速读取模式
	return r.randomRead(r.Fh, d, bias, key)
}

var D []byte = make([]byte, 1372)

// readfile
// 	随机读取，适配最后一包
func (r *Rd) randomRead(fh *os.File, d []byte, bias int64, key []byte) ([]byte, int64, bool, error) {
	if err = r.init(); err != nil {
		return nil, 0, false, err
	}

	_, err := fh.ReadAt(d, bias)
	if err != nil {
		if err == io.EOF {
			if r.fs-bias <= 1 {
				d = nil
				return nil, 0, true, nil
			}
			d = make([]byte, r.fs-bias, r.fs-bias+9)
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
	rbias    int64
}

// init 初始化函数
func (w *Wt) init() {
	if !w.initflag {

		w.bs = 4194304
		w.block = make([]byte, w.bs)
		w.rang = [2]int64{
			0, w.bs,
		}
		w.initflag = true
	}
}

// WriteFile 写入文件
//  传入参数: 原始数据, 偏置, 是否清空缓存(最后数据)
//  块中数据不连续也会被写入
func (w *Wt) WriteFile(d []byte, bias int64, end bool) error {

	w.init()

	if true {
		_, err = w.Fh.WriteAt(d, bias)
		return err
	}

	if bias < w.rang[0] { // 非缓存范围 直接写入
		_, err = w.Fh.WriteAt(d, bias)
		return err
	}

	l := int64(len(d))

	if w.rang[0]+w.bs < bias+l { //清空缓存块 w.rang[0]+w.bs-1 < bias+l-1
		_, err = w.Fh.WriteAt(w.block[:w.rang[1]-w.rang[0]+1], w.rang[0])

		// 重置
		w.rang[0] = bias
	}
	//存入缓存块
	copy(w.block[bias-w.rang[0]:], d)
	w.rang[1] = bias + l - 1

	if end { // 清空缓存
		w.Fh.WriteAt(w.block[:w.rang[1]-w.rang[0]+1], w.rang[0])

		// 之后的数据都不会再使用缓存块了
		w.rang[0] = w.rang[1] + 1
	}

	return err
}
