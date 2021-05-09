package crypter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"strconv"
)

/* 加密相关 */
var err error

// CbcEncrypt AES CBC加密
func CbcEncrypt(key []byte, p []byte) error {
	if len(key) != 16 {
		return errors.New("the secret key's length != 16")
	}

	if len(p)%16 != 0 {
		return errors.New("plaintext's length isn't integer multiples of 16，length：" + strconv.Itoa(len(p)))
	}
	block, err := aes.NewCipher(key) //key
	if err != nil {
		return err
	}

	mode := cipher.NewCBCEncrypter(block, key) // key is also used as the initialization vector
	mode.CryptBlocks(p[0:], p)

	return nil
}

// CbcDecrypt AES CBC解密
func CbcDecrypt(key []byte, c []byte) error {
	lenData := len(c)

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	if lenData%16 != 0 {
		err := errors.New("the data's length != 16*n; length is " + strconv.Itoa(lenData))
		if err != nil {
			return err
		}
	}
	mode := cipher.NewCBCDecrypter(block, key)
	mode.CryptBlocks(c[0:], c)

	return nil
}

// GenRsaKey 生成RSA 512的私钥(不定长度)和公钥(162)
func RsaGenKey() ([]byte, []byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, nil, err
	}
	privateKey := x509.MarshalPKCS1PrivateKey(key)

	publicKey, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	return privateKey, publicKey, nil
}

// RsaEncrypt RSA 加密
func RsaEncrypt(data, publicKey []byte) ([]byte, error) {

	pubInterface, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	// ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, pub, data)
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubInterface.(*rsa.PublicKey), data, nil)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// RsaDecrypt RSA 解密
func RsaDecrypt(ciphertext, privateKey []byte) ([]byte, error) {

	priv, err := x509.ParsePKCS1PrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	data, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}
