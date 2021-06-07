package cmd

import (
	"btest/com"
	"btest/setting"
	"btest/utils"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"

	cli "github.com/urfave/cli/v2"
)

var err error

func Run(c *cli.Context) error {

	if c.NArg() > 0 {

		if c.NArg() == 1 {

			if c.Args().Get(0) == "version" { // version
				fmt.Println("sudp version 0.1.0 Windows ")
			} else if c.Args().Get(0) == "init" { // 初始化所有设置
				if err = setting.Init(); err != nil {
					fmt.Println(err.Error())
				} else {
					fmt.Println("初始化设置成功！")
				}
			} else if c.Args().Get(0) == "show" { // 打印当前设置
				var s *setting.S
				if s, err = setting.Read(); err != nil {
					fmt.Println(err.Error())
				} else {
					// fmt.Printf("%+v\n", *s)
					if r, err := json.MarshalIndent(*s, "", "\t"); err != nil {
						fmt.Println(err)
					} else {
						fmt.Println(string(r))
					}
				}
			}

		} else if c.NArg() == 2 {

			if c.Args().Get(0) == "upload" { // upload
				path := c.Args().Get(1)
				if !com.Exist(path) {
					fmt.Println("路径不存在")
				} else {
					LanIP, err := com.GetLanIP()
					if err == nil {
						fmt.Println(LanIP)
						fmt.Println("")

						fmt.Println(utils.UpLoad(path))

					} else {
						if strings.Contains(err.Error(), "unreachabl") {
							fmt.Println("无网络")
						} else {
							fmt.Println(err)
						}
					}
				}

			} else if c.Args().Get(0) == "download" { // download
				var p string = c.Args().Get(1)
				var port int = 0
				var raddrIP net.IP
				if strings.Contains(p, ":") {
					if port, err = strconv.Atoi(strings.Split(p, ":")[1]); err != nil {
						fmt.Println("不正确的端口")
					}
					raddrIP = net.ParseIP(strings.Split(p, ":")[0])
					if raddrIP == nil {
						fmt.Println("不正确的IP")
					}
				} else {
					raddrIP = net.ParseIP(p)
					if raddrIP == nil {
						fmt.Println("不正确的IP")
					}
				}
				fmt.Println(utils.DownLoad(raddrIP, port))

			} else if c.Args().Get(0) == "MTU" { // MTU
				var newMTU int
				if newMTU, err = strconv.Atoi(c.Args().Get(1)); err != nil {
					fmt.Println("错误", err)
				}
				if newMTU > 500 && newMTU < 65500 {
					if err = setting.SetMTU(newMTU); err != nil {
						fmt.Println(err)
					}
				} else {
					fmt.Println("MTU范围不合理")
				}

			} else if c.Args().Get(0) == "auth" { // Auth
				if err = setting.SetAuth([]byte(c.Args().Get(1))); err != nil {
					fmt.Println(err)
				}

			} else if c.Args().Get(0) == "crypt" { // Crypt
				var e bool
				if c.Args().Get(1) == "true" {
					e = true
				} else if c.Args().Get(1) == "false" {
					e = false
				} else {
					fmt.Println("无效参数")
				}
				if err = setting.SetCrypt(e); err != nil {
					fmt.Println(err)
				}

			} else if c.Args().Get(0) == "storePath" { // StorePath

				path := c.Args().Get(1)
				if err = com.FloderExist(path); err != nil {
					fmt.Println("设置失败", err.Error())
				} else {
					if err = setting.SetStorePath(path); err != nil {
						fmt.Println(err)
					}
				}

			} else if c.Args().Get(0) == "LIP" { // LIP
				if err = setting.SetLIP(c.Args().Get(1)); err != nil {
					fmt.Println(err)
				}

			} else if c.Args().Get(0) == "LPort" { // LPort
				var port int
				if port, err = strconv.Atoi(c.Args().Get(1)); err != nil {
					fmt.Println(err)
				} else {
					if err = setting.SetLPort(port); err != nil {
						fmt.Println(err)
					}
				}

			} else if c.Args().Get(0) == "RPort" { // RPort
				var port int
				if port, err = strconv.Atoi(c.Args().Get(1)); err != nil {
					fmt.Println(err)
				} else {
					if err = setting.SetRPort(port); err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("此设置只对下载有效，确保和对方匹配！")
					}
				}
			}
		}

	} else {
		fmt.Println("SUDP is a file transfer protocol based on UDP, more info https://github.com/lysShub/sudp")

	}

	// name := "world"
	// if c.NArg() > 0 {
	// 	name = c.Args().Get(0)
	// }

	// if c.String("lang") == "english" {
	// 	fmt.Println("hello", name)
	// } else {
	// 	fmt.Println("你好", name)
	// }
	return nil
}
