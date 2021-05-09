package packet

import (
	"errors"
	"hash/crc32"
	"sudp/internal/crypter"
)

// PackageDataPacket 打包为数据包
//	 参数: d:原始数据; b:偏置; key:密钥; final:最后一个数据包
//	 返回：打包后数据包，原始数据长度, 是否最后一个数据包
func PackageDataPacket(d []byte, b int64, key []byte, final bool) ([]byte, int64, bool, error) {
	var dl int64 = int64(len(d))

	// data bias
	for i := uint8(0); i < 4; i++ {
		d = append(d, uint8(b>>(8*(4-i)-2)))
	}
	d = append(d, uint8(b<<2))

	// end
	if final {
		d[len(d)-1] = d[len(d)-1] + 0b10
	}

	// CRC IEEE
	s := crc32.ChecksumIEEE(d)
	d = append(d, uint8(s), uint8(s>>8), uint8(s>>16), uint8(s>>24))

	// 加密
	if key != nil {

		var l uint8 = uint8(16 - len(d)%16)
		for i := uint8(0); i < l; i++ {
			d = append(d, uint8(l))
		}

		if err := crypter.CbcEncrypt(key[:16], d); err != nil { // encrypto
			d = nil
			return nil, dl, final, err
		}
	}

	return d, dl, final, nil

}

// ParseDataPacket parse data packet
//	 返回: 原始数据长度; 偏置; 是否最后包
//	 由于解包不会追加数据, 所以d的形参和实参是相同的, 如果解包没有错误, d[:原始数据长度]即可获得原始数据
func ParseDataPacket(d []byte, k []byte) (int64, int64, bool, error) {

	var pl int // 原始数据包长度

	// 解密
	if k != nil && len(k) >= 16 {
		if err := crypter.CbcDecrypt(k[:16], d); err != nil {
			return 0, -1, false, err
		}
		pl = len(d) - int(d[len(d)-1])
	} else {
		pl = len(d)
	}

	// check sum
	rS := crc32.ChecksumIEEE(d[:pl])
	if rS != 0x2144DF1C {
		err := errors.New("CRC verify failed")
		return 0, -1, false, err
	}

	var end bool = false
	if 0b10&d[pl-5] == 2 {
		end = true
	}

	// bias
	var b int64 = int64(d[pl-9])<<30 + int64(d[pl-8])<<22 + int64(d[pl-7])<<14 + int64(d[pl-6])<<6 + int64(d[pl-5])>>2

	return int64(pl) - 9, b, end, nil
}
