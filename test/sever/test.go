package main

import (
	"fmt"
	"net"
	"sudp/tasker"
)

func main() {

	// 发送
	t := new(tasker.Tasker)
	t.Addr = &net.UDPAddr{IP: net.ParseIP("192.168.0.50"), Port: 19986} //192.168.0.50
	t.Encrypto = false
	// t.Path = `/root/workspace/sudp/tmp/r/DSPdsign.7z`
	t.Path = `/root/workspace/sudp/tmp/r/tele.apk`

	fmt.Println(t.Send(func(requestBody []byte) bool {
		return true
	}))

}
