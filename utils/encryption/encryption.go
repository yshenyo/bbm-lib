package encryption

import (
	"bytes"
	"crypto/des"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const BASE64Table = "GT9PyaGWUBNxfCdaPmpFdG9Ln6yjUHCAf7BX5fygEQmcx3W5wpgJsEcS3y7fqSLB"

func Encode(data string) (str string, err error) {
	// content := *(*[]byte)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&data))))
	// coder := base64.NewEncoding(BASE64Table)
	// return coder.EncodeToString(content)
	key := []byte("Bw5dCgG8")
	strEncrypted, err := Encrypt(data, key)
	if err != nil {
		return "", err
	}
	return strEncrypted, nil
}

func EncodeWithCheck(data string) (str string, err error) {
	_, err = Decode(data)
	if err == nil {
		return data, nil
	}
	str, err = Encode(data)
	return
}

func PasswordDecode(data string) string {
	str, err := Decode2(data, true)
	if err != nil {
		return data
	}
	return str
}

func PasswordEncode(data string) (str string, err error) {
	return EncodeWithCheck(data)
}

func Decode(data string) (str string, err error) {
	// coder := base64.NewEncoding(BASE64Table)
	// result, _ := coder.DecodeString(data)
	// return *(*string)(unsafe.Pointer(&result))
	return Decode2(data, true)
}

func Decode_Nologging(data string) (str string, err error) {
	// coder := base64.NewEncoding(BASE64Table)
	// result, _ := coder.DecodeString(data)
	// return *(*string)(unsafe.Pointer(&result))
	return Decode2(data, false)
}

func Decode2(data string, isLogErr bool) (str string, err error) {
	key := []byte("Bw5dCgG8")
	strDecrypted, err := Decrypt(data, key)
	if err != nil {
		return "", err
	}
	return strDecrypted, nil

}
func ZeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

func ZeroUnPadding(origData []byte) []byte {
	return bytes.TrimFunc(origData,
		func(r rune) bool {
			return r == rune(0)
		})
}

func Encrypt(text string, key []byte) (string, error) {
	src := []byte(text)
	block, err := des.NewCipher(key)
	if err != nil {
		return "", err
	}
	bs := block.BlockSize()
	src = ZeroPadding(src, bs)
	if len(src)%bs != 0 {
		return "", errors.New("need a multiple of the block size")
	}
	out := make([]byte, len(src))
	dst := out
	for len(src) > 0 {
		block.Encrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	return hex.EncodeToString(out), nil
}

func Decrypt(decrypted string, key []byte) (string, error) {
	src, err := hex.DecodeString(decrypted)
	if err != nil {
		return "", err
	}
	block, err := des.NewCipher(key)
	if err != nil {
		return "", err
	}
	out := make([]byte, len(src))
	dst := out
	bs := block.BlockSize()
	if len(src)%bs != 0 {
		return "", errors.New("crypto/cipher: input not full blocks")
	}
	for len(src) > 0 {
		block.Decrypt(dst, src[:bs])
		src = src[bs:]
		dst = dst[bs:]
	}
	out = ZeroUnPadding(out)
	return string(out), nil
}

func PasswordMd5(password string) (p string) {
	pass := md5.Sum([]byte(password[0:1]))
	pStr := fmt.Sprintf("%x", pass)
	word := md5.Sum([]byte(password[1:]))
	wStr := fmt.Sprintf("%x", word)
	p = strings.ToUpper(pStr[0:16] + wStr[16:])
	return p
}
