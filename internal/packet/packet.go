package packet

import (
	"bytes"
	"errors"
	"hash/crc32"

	"sudp/internal/crypter"
)

// PackagePacket 打包为协议包
//	 参数: d:原始数据; b:偏置; key:密钥; final:最后一个数据包
//	 返回：打包后数据包，原始数据长度, 是否最后一个数据包
func PackagePacket(d []byte, b int64, key []byte, final bool) ([]byte, int64, bool, error) {
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

// ParsePacket 协议包解包
//	 返回: 原始数据长度; 偏置; 是否最后包
//	 由于解包不会追加数据, 所以d的形参和实参是相同的, 如果解包没有错误, d[:原始数据长度]即可获得原始数据
func ParsePacket(d []byte, k []byte) (int64, int64, bool, error) {

	var pl int // 原始数据包长度

	// 解密
	if len(k) >= 16 {
		if err := crypter.CbcDecrypt(k[:16], d); err != nil {
			return 0, -1, false, err
		}

		if 0 < d[len(d)-1] || d[len(d)-1] < 16 {
			pl = len(d) - int(d[len(d)-1])
			var fl uint8 = d[len(d)-1]
			for _, v := range d[pl:] {
				if v != fl {
					pl = len(d)
					break
				}
			}
		} else {
			pl = len(d)
		}

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

// SecureEncrypt 安全加密
//  前16B的密钥会自动添加
func SecureEncrypt(d []byte, k []byte) ([]byte, error) {

	if len(k) < 16 {
		return nil, errors.New("invalid k")
	}
	var tk [16]byte
	copy(tk[:], k[:16])
	d = append(tk[:], d...)

	var l uint8 = uint8(16 - len(d)%16)
	for i := uint8(0); i < l; i++ {
		d = append(d, l) // 填充
	}

	if err := crypter.CbcEncrypt(k[:16], d); err != nil { // encrypto
		d = nil
		return nil, err
	}
	return d, nil
}

// SecureDecrypt 解密安全加密
//  返回数据不包括密钥,如果密钥不符或出错会返回错误
func SecureDecrypt(d []byte, k []byte) ([]byte, error) {
	if len(k) < 16 {
		return nil, errors.New("invalid k")
	}

	var pl int // 原始数据长度

	if err := crypter.CbcDecrypt(k[:16], d); err != nil {
		return nil, err
	}

	if d[len(d)-1] > 0 || d[len(d)-1] < 16 {
		pl = len(d) - int(d[len(d)-1])
		var fl uint8 = d[len(d)-1]
		for _, v := range d[pl:] {
			if v != fl {
				pl = len(d)
				break
			}
		}
	} else {
		pl = len(d)
	}

	d = d[:pl]

	if bytes.Equal(k[:16], d[:16]) {
		return d[16:], nil
	}
	return nil, errors.New("invalid d")
}
