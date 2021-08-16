package main

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
)

func main() {
	ulit(`E:\浏览器下载\china_ip_list-master\china_ip_list-master\china_ip_list.txt`, `D:\Desktop\ips.txt`)
}

// 处理 https://github.com/17mon/china_ip_list 的数据
// 网段处理为IP, 主机号随机
func ulit(src, dst string) {

	fh, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	rh := bufio.NewScanner(fh)

	wh, err := os.OpenFile(dst, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	var f = func() string {
		r, err := rand.Int(rand.Reader, big.NewInt(255))
		if err != nil {
			panic(err)
		}
		return r.String()
	}

	for rh.Scan() {
		r := strings.Split(rh.Text(), `/`)
		if len(r) >= 2 {
			var ip, snet = r[0], r[1]
			var net int
			if net, err = strconv.Atoi(snet); err != nil {
				continue
			}
			var ips = strings.Split(ip, `.`)
			if len(ips) < 4 {
				continue
			}
			if net <= 8 {
				ips[3] = f()
			} else if net <= 16 {
				ips[2], ips[3] = f(), f()
			} else if net <= 24 {
				ips[1], ips[2], ips[3] = f(), f(), f()
			} else {
				continue
			}
			wh.Write([]byte(strings.Join(ips, `.`) + fmt.Sprintln()))

		}
	}

}
