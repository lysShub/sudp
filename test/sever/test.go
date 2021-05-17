package main

import (
	"fmt"
	"net"

	"github.com/lysShub/sudp"
)

func main() {

	// 发送 电脑
	w, err := sudp.NewWrite(func(r *sudp.Write) *sudp.Write {
		r.Laddr = &net.UDPAddr{IP: nil, Port: 19986} // HW st net.ParseIP("192.168.0.50")
		// r.Path = `/mnt/sdcard/a/Telegram.apk`
		r.Path = `../../tmp/r/Telegram.apk`
		// r.Path = `C:\Users\LYS\Desktop\Telegram.apk`
		return r
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(w.Write(func(requestBody []byte) bool { return true }))

	// fmt.Println(s.ReceiveHandshake(&net.UDPAddr{IP: net.ParseIP("192.168.0.50"), Port: 19986}, func(requestBody []byte) bool { return true }))
}

// 本机
func main2() {
	w, err := sudp.NewWrite(func(r *sudp.Write) *sudp.Write {
		r.Laddr = &net.UDPAddr{IP: net.ParseIP("10.8.145.88"), Port: 19987}
		r.Path = `../../tmp/r/Telegram.apk`
		return r
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(w.Write(func(requestBody []byte) bool { return true }))
}

// 虚拟机
func main3() {
	w, err := sudp.NewWrite(func(r *sudp.Write) *sudp.Write {
		r.Laddr = &net.UDPAddr{IP: nil, Port: 19986}
		r.Path = `../../tmp/r/Telegram.apk`
		return r
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(w.Write(func(requestBody []byte) bool { return true }))
}

func main1() {
	// http.ListenAndServe(":8000", http.FileServer(http.Dir("../../tmp/r/")))
	// return

	// // 发送
	// t := new(tasker.Tasker)
	// t.Addr = &net.UDPAddr{IP: net.ParseIP("192.168.0.50"), Port: 19986} //192.168.0.50
	// t.Encrypto = false
	// t.Path = `../../tmp/r/Telegram.apk`

	// fmt.Println(t.Send(func(requestBody []byte) bool {
	// 	return true
	// }))

}
