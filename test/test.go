package main

import (
	"io/fs"
	"path/filepath"
)

// var key = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf}
var key []byte = nil

var blockSize int = 1204

func main() {
	var floder string = `D:\Desktop\tieba` //`E:\浏览器下载`

	filepath.WalkDir(floder, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Type().IsRegular() {
			if err = action(path); err != nil {
				panic(err)
			}
		}
		return nil
	})
}
