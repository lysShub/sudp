package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"sudp/internal/file"
	"sudp/internal/packet"
	"time"
)

func main() {

	fh, err := os.Open(`D:\a.mkv`)
	if err != nil {
		fmt.Println(err)
		return
	}

	wh, err := os.OpenFile(`C:\b.mkv`, os.O_CREATE|os.O_WRONLY, 0666)
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

	var bias int64
	var count int64 = 0
	go func() {
		for {
			time.Sleep(time.Second)
			fmt.Println(count >> 20) // MB/s  , bias>>30
			count = 0
		}
	}()

	// 初始化完成
	d := make([]byte, 1370, 1420)

	for bias = int64(0); ; {
		p, dl, end, err := r.ReadFile(d, bias, key)
		if err != nil {
			fmt.Println(err)
			return
		}

		count += dl

		/* 读取完成 */
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
			fmt.Println("耗时", b-a, "速度", 7642/(b-a))
			fmt.Println(bias)
			return
		}
		bias = bias + dl
	}

}

// 比较两个文件的差异
func main1() {
	fh, err := os.Open(`D:\a.mkv`)
	if err != nil {
		fmt.Println(err)
		return
	}
	fh2, err := os.Open(`C:\b.mkv`)
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
					fmt.Println("第" + strconv.Itoa(int(i)+j) + "位不相同")
					break
				}
			}

			b, b1 := make([]byte, 100), make([]byte, 100)
			_, err := fh.ReadAt(b, int64(f))
			if err != nil {
				fmt.Println(err)
				return
			}
			_, err = fh2.ReadAt(b1, int64(f))
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
