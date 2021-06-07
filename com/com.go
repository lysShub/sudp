package com

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

// GetLanIP 获取网络号：NetIP ,用于判断是否在局域网中
func GetLanIP() (net.IP, error) {
	conn, err := net.Dial("udp", "3.3.3.3:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var LanIP = conn.LocalAddr().(*net.UDPAddr).IP
	return LanIP, nil

	ifaces, err := net.Interfaces()
	if err != nil || len(ifaces) == 0 {
		return nil, errors.New("no network card discover")
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if tIP, netIP, err := net.ParseCIDR(a.String()); err == nil {
				if LanIP.Equal(tIP) {
					return netIP.IP, nil
				}
			}
		}
	}
	return nil, errors.New("unknown error")
}

func Exist(s string) bool {
	fi, err := os.Stat(s)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		var ts int64
		filepath.Walk(s, func(path string, info os.FileInfo, err error) error {
			ts += info.Size()
			return nil
		})
		if ts == 0 {
			return false
		}

	} else {
		fh, err := os.Open(s)
		if err != nil {
			return false
		}
		defer fh.Close()
	}
	return true
}

func FloderExist(s string) error {
	fi, err := os.Stat(s)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("路径不存在")
		}
		return err
	}
	if fi.IsDir() {
		return nil
	} else {
		return errors.New("路径不是文件夹")
	}
}

func GetS(n int, char string) (s string) {
	for i := 1; i <= n; i++ {
		s += char
	}
	return
}

func FormatSpeed(s int) string {
	if s < 1204 {
		return strconv.Itoa(s) + " B/s"
	} else if s > 1024 && s < 1024*1204 {

		return fmt.Sprintf("%.2f", float64(s)/1024) + " KB/s"
	} else if s > 1024*1024 && s < 1024*1024*1024 {
		return fmt.Sprintf("%.2f", float64(s)/(1024*1024)) + " MB/s"
	} else {
		return fmt.Sprintf("%.2f", float64(s)/(1024*1024*1024)) + " GB/s"
	}
}
