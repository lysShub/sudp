package cmd

import "github.com/urfave/cli/v2"

var Par []cli.Flag = []cli.Flag{
	&cli.StringFlag{ // version
		Name:  "version",
		Usage: "version",
	},
	&cli.StringFlag{
		Name:  "upload",
		Usage: "发送文件; sudp upload 文件(夹)路径",
	},
	&cli.StringFlag{
		Name:  "download",
		Usage: "接收文件; sudp download 对方IP",
	},
	&cli.StringFlag{
		Name:  "MTU",
		Usage: "重置MTU",
	},
	&cli.StringFlag{
		Name:  "auth",
		Usage: "设置权鉴",
	},
	&cli.StringFlag{
		Name:  "crypt",
		Usage: "设置加密, true/false",
		Value: "true",
	},
	&cli.StringFlag{
		Name:  "storePath",
		Usage: "设置储存路径",
		Value: "当前路径",
	},
	&cli.StringFlag{
		Name:  "LIP",
		Usage: "设置本地IP, 多网卡下指定网卡时使用",
	},

	&cli.StringFlag{
		Name:  "LPort",
		Usage: "设置本地IP",
		Value: "19986",
	},
	&cli.StringFlag{
		Name:  "RPort",
		Usage: "设置对方IP, 仅在下载时生效",
		Value: "19986",
	},
}
