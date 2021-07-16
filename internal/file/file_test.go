package file_test

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/lysShub/sudp/internal/file"
	"github.com/lysShub/sudp/internal/packet"
)

// 覆盖测试路径 。将读写此路径下所有文件, 选择一个文件较多的无关紧要的路径
var floder string = `E:\浏览器下载`

var key []byte = nil
var blockSize int = 1024

func TestCover(t *testing.T) {
	// 覆盖测试 go test -run="Cover"

	var fileSizes []int = []int{
		// 0, 1, blockSize - 1, blockSize, blockSize + 1, 4194304 - 1, 4194304, 4194304 + 1, 4194304*8 - 1, 4194304 * 8, 4194304*8 + 1,
		4194304 * 4,
	}
	// for i := 0; i < 10; i++ {
	// 	fileSizes = append(fileSizes, randInt(1024*1024*100))
	// }

	var path string = `./test.file`
	for _, v := range fileSizes {
		creat(path, v)

		if err := action(path); err != nil {
			panic(err)
		}

		os.Remove(path)
	}
}

// action 读写文件
func action(path string) error {
	fh, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fh.Close()

	var path2 = path + ".copy"
	wh, err := os.OpenFile(path2, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	defer func() {
		wh.Close()
		os.Remove(path2)
	}()

	r, err := file.NewRead(fh)
	if err != nil {
		panic(err)
	}
	w := new(file.Wt)
	w.Fh = wh

	var bias int64 = 0
	var da = make([]byte, blockSize)

	for bias = int64(0); ; {
		p, dl, end, err := r.ReadFile(da, bias, key)
		if err != nil {
			panic(err)
		}

		dl2, bias2, end2, err := packet.ParsePacket(p, key)
		if err != nil {
			panic(err)
		}
		if dl != dl2 || end != end2 || bias != bias2 {
			fmt.Println(dl, dl2, end, end2, bias, bias2)
			panic("解包失败")
		}

		// 写入
		if err = w.WriteFile(p[:dl2], bias2, end2); err != nil {
			panic(err)
		}
		bias = bias + dl

		if end {

			// 检查
			if !check(fh, wh) {
				return errors.New("不相同" + path)
			}
			return nil

		}
	}
}

// creat 生成一个文件
func creat(path string, fileSize int) {
	var size int = 1024 * 16

	var block []byte = make([]byte, size)
	for i := 0; i < size; i++ {
		block[i] = byte(i)
	}
	wh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer wh.Close()

	for i := 0; i < fileSize/size; i++ {
		if _, err = wh.Write(block); err != nil {
			panic(err)
		}
	}
	if _, err = wh.Write(block[:fileSize%size]); err != nil {
		panic(err)
	}
}

// check
func check(fh1, fh2 *os.File) bool {

	h1 := md5.New()
	io.Copy(h1, fh1)
	s1 := h1.Sum(nil)

	h2 := md5.New()
	io.Copy(h2, fh2)
	s2 := h2.Sum(nil)

	if bytes.Equal(s1, s2) {
		return true
	} else {
		// 定位

		r1 := bufio.NewReader(fh1)
		r2 := bufio.NewReader(fh2)

		var da1, da2 []byte = make([]byte, 1024), make([]byte, 1024)
		var n1, n2 int
		for {
			n2, _ = r2.Read(da2)
			n1, _ = r1.Read(da1)

			if !bytes.Equal(da1[:n1], da2[:n2]) {
				for i, v := range da1 {
					if v != da2[i] {
						fi, _ := fh1.Stat()
						fmt.Println(fh1.Name(), fi.Size())
						fmt.Println(da1[i:])
						fmt.Println(da2[i:])
						return false
					}
				}
			}
			if n1 < len(da1) || n2 < len(da2) {
				return true
			}
		}
	}
}

func randInt(max int) int {

	r, err := rand.Int(rand.Reader, new(big.Int).SetInt64(int64(max)))
	if err != nil {
		return int(time.Now().Unix())
	} else {
		return int(r.Int64())
	}
}
