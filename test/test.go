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
)

func main() {
	// 接受方
	r, err := sudp.NewRead(func(r *sudp.Read) *sudp.Read {
		r.Raddr = &net.UDPAddr{IP: net.ParseIP("119.3.166.124"), Port: 19986} // HW st
		r.Path = `D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp`
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

// 本机
func main2() {
	r, err := sudp.NewRead(func(r *sudp.Read) *sudp.Read {
		r.Raddr = &net.UDPAddr{IP: net.ParseIP("10.8.145.88"), Port: 19987}
		r.Path = `D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp`
		return r
	})

	if err != nil {
		fmt.Println(err)
		return
	}
	a := time.Now()
	fmt.Println(r.Read(nil))
	fmt.Println("耗时", time.Now().Sub(a))
}

// 虚拟机
func main3() {
	r, err := sudp.NewRead(func(r *sudp.Read) *sudp.Read {
		r.Raddr = &net.UDPAddr{IP: net.ParseIP("192.168.43.183"), Port: 19986}
		r.Path = `D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp`
		return r
	})

	if err != nil {
		fmt.Println(err)
		return
	}
	a := time.Now()
	fmt.Println(r.Read(nil))
	fmt.Println("耗时", time.Now().Sub(a))
}

func main1() {
	difference()
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
			return
		}

	}
}
