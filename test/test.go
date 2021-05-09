package main

import (
	"bytes"
	"fmt"
	"net"
	_ "net/http/pprof"
	"os"
	"strconv"
	"sudp"
)

func main() {

	r, err := sudp.NewRead(func(r *sudp.Read) *sudp.Read {
		r.Raddr = &net.UDPAddr{IP: net.ParseIP("119.3.166.124"), Port: 19986}
		r.Path = `D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp`
		return r
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(r.Read(nil))

	// fmt.Println(s.SendHandshake(&net.UDPAddr{IP: nil, Port: 19986}, &net.UDPAddr{IP: net.ParseIP("119.3.166.124"), Port: 19986}, nil))

}

func main2() {

	// 172.30.0.1:49529

	// t := new(tasker.Tasker)
	// t.Addr = &net.UDPAddr{IP: net.ParseIP("119.3.166.124"), Port: 19986} // net.ParseIP("172.30.0.1")
	// t.Path = `D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp`

	// fmt.Println(t.Receive(&net.UDPAddr{IP: nil, Port: 19986}, []byte("11")))
}

func main1() {
	// // &net.UDPAddr{IP: net.ParseIP("114.116.254.26"), Port: 19986},
	// // 收 119.3.166.124

	// go func() {
	// 	t := new(tasker.Tasker)
	// 	t.Addr = &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 19986} //172.30.1.211
	// 	t.Path = `D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp`

	// 	fmt.Println(t.Receive(&net.UDPAddr{IP: nil, Port: 19986}, []byte("11")))
	// }()

	// http.ListenAndServe(":8080", nil)
}

// 比较两个文件的差异
func difference() {
	fh, err := os.Open(`D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp\DSPdsign.7z`)
	if err != nil {
		fmt.Println(err)
		return
	}
	fh2, err := os.Open(`D:\OneDrive\code\go\src\github.com\lysShub\sudp\tmp\r\DSPdsign.7z`)
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
