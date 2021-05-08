package main

import (
	"fmt"
	"net"
	"sudp"
	"time"
)

func main() {
	s := new(sudp.SUDP)
	s.Encrypt = true
	s.MTU = 1237
	s.TimeOut = time.Second

	fmt.Println(s.ReceiveHandshake(&net.UDPAddr{IP: net.ParseIP("192.168.0.50"), Port: 19986}, func(requestBody []byte) bool { return true }))
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
