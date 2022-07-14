package bbGoogleAuth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"fmt"
	"strings"
	"time"
)

// GetSecret
// 随机生成16位的秘钥（32位也支持，我们这边只采用16位）
// 存在秘钥重复问题，需要在使用时校验， 秘钥字符只包含字母A-Z，数字2-7
// @return 随机生成的16位秘钥
func GetSecret() string {
	randomStr := randStr(16)
	return strings.ToUpper(randomStr)
}

// VerifyCode
// 通过秘钥校验验证码
// @param secret string 16位的秘钥
// @param code int32 google验证器上当前时间的验证码
// @return bool 验证码是否正确
// @return error 秘钥输入错误，无法解码
func VerifyCode(secret string, code int32) (bool, error) {
	codeSet, err := getCode(secret, 0)
	if err != nil {
		return false, err
	}
	for _, v := range codeSet {
		if v == code {
			return true, nil
		}
	}
	return false, nil
}

func GetSecretQrcode(user string) string {
	return fmt.Sprintf("otpauth://totp/%s?secret=%s", user, GetSecret())
}

func randStr(strSize int) string {
	dictionary := "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	var bytes = make([]byte, strSize)
	_, _ = rand.Read(bytes)
	for k, v := range bytes {
		bytes[k] = dictionary[v%byte(len(dictionary))]
	}
	return string(bytes)
}

func getCode(secret string, offset int64) ([]int32, error) {
	key, err := base32.StdEncoding.DecodeString(secret)
	//检验Base32解码报错，只支持字母A-Z，数字2-7
	if err != nil {
		return nil, err
	}

	//生成一次性密码，间隔30s，同时生成当前时间的前30s和后30s
	epochSeconds := time.Now().Unix() + offset
	var codeSet []int32
	codeSet = append(codeSet, int32(oneTimePassword(key, toBytes(epochSeconds/30))))
	codeSet = append(codeSet, int32(oneTimePassword(key, toBytes((epochSeconds+int64(30))/30))))
	codeSet = append(codeSet, int32(oneTimePassword(key, toBytes((epochSeconds-int64(30))/30))))
	return codeSet, nil
}

func oneTimePassword(key []byte, value []byte) uint32 {
	// 签署所使用的方法是HMAC-SHA1（哈希运算消息认证码），
	//以一个密钥和一个消息为输入，生成一个消息摘要作为输出。
	//用共享密钥做为secret，时间戳/30做为输入值来生成20字节的SHA1值
	hmacSha1 := hmac.New(sha1.New, key)
	hmacSha1.Write(value)
	hash := hmacSha1.Sum(nil)
	offset := hash[len(hash)-1] & 0x0F

	// 转化为标准的32bit无符号整数
	hashParts := hash[offset : offset+4]
	hashParts[0] = hashParts[0] & 0x7F
	number := toUint32(hashParts)

	// 通过1000,000取整，获得6位密码
	pwd := number % 1000000
	return pwd
}

func toUint32(bytes []byte) uint32 {
	return (uint32(bytes[0]) << 24) + (uint32(bytes[1]) << 16) +
		(uint32(bytes[2]) << 8) + uint32(bytes[3])
}

func toBytes(value int64) []byte {
	var result []byte
	mask := int64(0xFF)
	shifts := [8]uint16{56, 48, 40, 32, 24, 16, 8, 0}
	for _, shift := range shifts {
		result = append(result, byte((value>>shift)&mask))
	}
	return result
}
