package main

import (
	"bytes"
	"fmt"
	"net"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/lysShub/sudp"
	"github.com/lysShub/sudp/internal/file"
	"github.com/lysShub/sudp/internal/packet"
)

func main() {
	// 接受方
	r, err := sudp.NewRead(func(r *sudp.Read) *sudp.Read {
		r.Raddr = &net.UDPAddr{IP: net.ParseIP("10.8.183.23"), Port: 19986} // HW st 119.3.166.124
		// r.Path = `D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp`
		r.Path = `./`
		return r
	})

	if err != nil {
		fmt.Println(err)
		return
	}
	a := time.Now()
	fmt.Println(r.Read(nil))
	fmt.Println("耗时", time.Now().Sub(a))

	// fmt.Println(s.SendHandshake(&net.UDPAddr{IP: nil, Port: 19986}, &net.UDPAddr{IP: net.ParseIP("119.3.166.124"), Port: 19986}, nil))

}

func main1() {
	difference()
}

//
func main2() {
	fh, err := os.Open(`C:\Users\LYS\Desktop\a.pkg`)
	if err != nil {
		fmt.Println(err)
		return
	}

	wh, err := os.OpenFile(`C:\Users\LYS\Desktop\b.pkg`, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Println(err)
		return
	}

	r := new(file.Rd)
	r.Fh = fh

	w := new(file.Wt)
	w.Fh = wh

	a := time.Now().Unix()
	key := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}
	key = nil

	// 初始化完成
	d := make([]byte, 1370, 1420)
	for bias := int64(0); ; {
		p, dl, end, err := r.ReadFile(d, bias, key)
		if err != nil {
			fmt.Println(err)
			return
		}

		/* 发送完成 */
		dl2, bias2, end2, err := packet.ParsePacket(p, key)
		if err != nil {
			fmt.Println(err)
			return
		}
		if dl != dl2 {
			fmt.Println("dl != dl2")
			return
		}
		if end != end2 {
			fmt.Println("end != end2")
			return
		}
		if bias != bias2 {
			fmt.Println("bias != bias2")
			return
		}

		// 写入
		if err = w.WriteFile(p[:dl2], bias2, end2); err != nil {
			fmt.Println(err)
			return
		}

		if end {
			b := time.Now().Unix()
			fmt.Println("耗时", b-a, "速度", 943/(b-a))
			fmt.Println(bias)
			return
		}
		bias = bias + dl
	}
}

// 比较两个文件的差异
func difference() {
	fh, err := os.Open(`D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp\Telegram.apk`)
	if err != nil {
		fmt.Println(err)
		return
	}
	fh2, err := os.Open(`D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp\r\Telegram.apk`)
	if err != nil {
		fmt.Println(err)
		return
	}

	fi, err := fh.Stat()
	if err != nil {
		fmt.Println(err)
		return
	}

	b, b1 := make([]byte, 1024), make([]byte, 1024)
	for i := int64(0); i < fi.Size(); i = i + 1024 {
		_, err := fh.ReadAt(b, i)
		if err != nil {
			fmt.Println(err)
			return
		}

		_, err = fh2.ReadAt(b1, i)
		if err != nil {
			fmt.Println(err)
			return
		}

		if !bytes.Equal(b, b1) {
			var f int
			for j, v := range b {
				if v != b1[j] {
					f = int(i) + j
					fmt.Println("第" + strconv.Itoa(f) + "位不相同")
					break
				}
			}

			b, b1 := make([]byte, 100), make([]byte, 100)
			_, err := fh.ReadAt(b, int64(f-5))
			if err != nil {
				fmt.Println(err)
				return
			}
			_, err = fh2.ReadAt(b1, int64(f-5))
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(b)
			fmt.Println(b1)

			fmt.Println("------------------------")
			da, tad := make([]byte, 10), make([]byte, 10)
			for i := f; i < int(fi.Size()); i += 1372 {
				if _, err = fh.ReadAt(da, int64(i)); err != nil {
					fmt.Println(err)
					return
				}
				if !bytes.Equal(da, tad) {
					fmt.Println("在第" + strconv.Itoa(i))
					fmt.Println(da)
					return
				}
			}
			return
		}

	}
}
