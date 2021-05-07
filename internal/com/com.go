package com

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gogs/chardet"
	"golang.org/x/net/html/charset"
)

// Writers file handles
var Writers []io.Writer

func init() {
	Writers = append(Writers, os.Stderr)
}

// Errorlog logger
func Errorlog(err ...error) bool {
	// writers = []io.Writer{
	// 	errLogHandle,
	// 	os.Stdout,
	// }
	var haveErr bool = false
	for i, e := range err {
		if e != nil {
			haveErr = true
			_, fp, ln, _ := runtime.Caller(1) //行数

			w := io.MultiWriter(Writers...)
			logger := log.New(w, "", log.Ldate|log.Ltime) //|log.Lshortfile
			logger.Println(fp + ":" + strconv.Itoa(ln) + "." + strconv.Itoa(i+1) + "==>" + e.Error())
		}
	}
	return haveErr
}

// ToUtf8 Convert to any encoding (as far as possible) to utf8 encoding
func ToUtf8(s []byte) []byte {

	// chardet echo charsets:Shift_JIS,EUC-JP,EUC-KR,Big5,GB18030,ISO-8859-2(windows-1250),ISO-8859-5,ISO-8859-6,ISO-8859-7,indows-1253,ISO-8859-8(windows-1255),ISO-8859-8-I,ISO-8859-9(windows-1254),windows-1256,windows-1251,KOI8-R,IBM424_rtl,IBM424_ltr,IBM420_rtl,IBM420_ltr,ISO-2022-JP

	d := chardet.NewTextDetector() //chardet is more precise charset.DetermineEncoding
	var rs *chardet.Result
	var err1, err2 error
	if len(s) > 1024 {
		if utf8.Valid(s[:1024]) {
			return s
		}
		rs, err1 = d.DetectBest(s[:1024])
	} else {
		if utf8.Valid(s) {
			return s
		}
		rs, err1 = d.DetectBest(s)
	}
	Errorlog(err1, err2)

	// charset input charsets:utf-8,ibm866,iso-8859-2,iso-8859-3,iso-8859-4,iso-8859-5,iso-8859-6,iso-8859-7,iso-8859-8,iso-8859-8-i,iso-8859-10,iso-8859-13,iso-8859-14,iso-8859-15,iso-8859-16,koi8-r,koi8-u,macintosh,windows-874,windows-1250,windows-1251,windows-1252,windows-1253,windows-1254,windows-1255,windows-1256,windows-1257,windows-1258,x-mac-cyrillic,gbk,gb18030,big5,euc-jp,iso-2022-jp,shift_jis,euc-kr,replacement,utf-16be,utf-16le,x-user-defined,

	var maps map[string]string = make(map[string]string)
	maps = map[string]string{
		"Shift_JIS":    "shift_jis",
		"EUC-JP":       "euc-jp",
		"EUC-KR":       "euc-kr",
		"Big5":         "big5",
		"GB18030":      "gb18030",
		"ISO-8859-2 ":  "iso-8859-2",
		"ISO-8859-5":   "iso-8859-5",
		"ISO-8859-6":   "iso-8859-6",
		"ISO-8859-7":   "iso-8859-7",
		"ISO-8859-8":   "iso-8859-8",
		"ISO-8859-8-I": "iso-8859-8-i",
		"ISO-8859-9":   "iso-8859-10",
		"windows-1256": "windows-1256",
		"windows-1251": "windows-1251",
		"KOI8-R":       "koi8-r",
		"ISO-2022-JP":  "iso-2022-jp",
		"UTF-16BE ":    "utf-16be",
		"UTF-16LE ":    "utf-16le",
	}

	ct := maps[rs.Charset]
	if ct == "" || err1 != nil { // use charset.DetermineEncoding
		_, name, b := charset.DetermineEncoding([]byte(s), "utf-8")
		if b {
			return s
		}
		ct = name
	}

	byteReader := bytes.NewReader(s)
	reader, err1 := charset.NewReaderLabel(ct, byteReader)
	r, err2 := ioutil.ReadAll(reader)

	if Errorlog(err1, err2) {
		return s
	}
	return r
}

// Wrap 各个系统下的换行符
func Wrap() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	} else if runtime.GOOS == "darwin" {
		return "\r"
	} else {
		return "\n"
	}
}

// GetNetIP 获取网络号：NetIP ,用于判断是否在局域网中
func GetNetIP() (net.IP, error) {
	conn, err := net.Dial("ip4:1", "114.114.114.114")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var LanIP net.IP
	var mask net.IPMask
	switch v := conn.LocalAddr().(type) {
	case *net.IPAddr:
		LanIP = v.IP
		mask = LanIP.DefaultMask()
	}
	if LanIP == nil || mask == nil {
		return nil, errors.New("unknown error")
	}
	return LanIP.Mask(mask), nil
}

// Info floder info
type Info struct {
	S []int64  //size 字节
	N []string //name 相对路径
	T []int64  //time 上次修改时间纳秒
}

// GetFloderInfo 获取文件夹信息
//  返回: 文件信息 根路径 排除文件路径
func GetFloderInfo(path string) (Info, string, []string, error) {
	var R Info
	var basePath string
	var outFile []string //

	fi, err := os.Stat(path)
	if err != nil {
		return R, "", nil, err
	}

	basePath = filepath.ToSlash(filepath.Dir(path)) + `/` //文件
	ISDIR := false
	if fi.IsDir() {
		ISDIR = true
		path = filepath.ToSlash(path) + `/`
		basePath = filepath.ToSlash(filepath.Dir(filepath.Dir(path))) + `/` //文件夹
	}

	rmap := make(map[string]int64) //
	tmap := make(map[string]int64) //
	var tp string
	if ISDIR {
		err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {

			if err != nil {
				if os.IsNotExist(err) {
					outFile = append(outFile, p)
					return nil
				} else if strings.Contains(err.Error(), `Access is denied.`) {
					outFile = append(outFile, p)
					return nil
				}
				return err
			}

			if info.IsDir() {
				return nil
			}
			if !info.Mode().IsRegular() {
				outFile = append(outFile, p)
				return nil
			}

			hl, err := os.Open(p)
			if err != nil {
				outFile = append(outFile, p)
				return nil
			}
			hl.Close()

			p, err = filepath.Rel(path, p)
			if err != nil {
				return err
			}
			tp = filepath.Base(path) + `/` + filepath.ToSlash(p)
			rmap[tp] = int64(info.Size())
			tmap[tp] = info.ModTime().UnixNano()
			return nil
		})
		if err != nil {
			return R, "", nil, err
		}
	} else {
		hl, err := os.Open(path)
		if err != nil {
			outFile = append(outFile, path)
			return R, "", nil, err
		}
		hl.Close()

		R.S = []int64{int64(fi.Size())}
		R.N = []string{filepath.Base(path)}
		R.T = []int64{int64(fi.ModTime().UnixNano())}
		return R, basePath, nil, nil
	}

	// sort
	var nameSlice []string
	for k := range rmap {
		if len(k) > 1024 {
			return R, "", nil, errors.New("file path too long: " + k)
		}
		nameSlice = append(nameSlice, k)
	}
	sort.Strings(sort.StringSlice(nameSlice))
	ls := len(nameSlice)

	sizeSlice := make([]int64, ls)
	timeSlice := make([]int64, ls)

	for i, v := range nameSlice {
		if i == 0 {
			sizeSlice[0] = rmap[v]
			timeSlice[0] = tmap[v]
		} else {
			sizeSlice[i] = rmap[v]
			timeSlice[i] = tmap[v]
		}
	}
	R.S = sizeSlice
	R.N = nameSlice
	R.T = timeSlice

	return R, basePath, outFile, nil
}
