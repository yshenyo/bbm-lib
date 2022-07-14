package captcha

import (
	"errors"
	"time"

	"github.com/go-redis/redis"

	"github.com/mojocn/base64Captcha"
)

type configJsonBody struct {
	Id            string
	CaptchaType   string
	VerifyValue   string
	DriverAudio   *base64Captcha.DriverAudio
	DriverString  *base64Captcha.DriverString
	DriverChinese *base64Captcha.DriverChinese
	DriverMath    *base64Captcha.DriverMath
	DriverDigit   *base64Captcha.DriverDigit
}

var store = base64Captcha.DefaultMemStore

func GenerateCaptcha(redis *redis.Client, imgHeight, imgWidth int, t time.Duration) (captchaId string, captchaImg string, err error) {
	var param configJsonBody
	param.DriverDigit = &base64Captcha.DriverDigit{
		Height:   imgHeight,
		Width:    imgWidth,
		Length:   4,
		MaxSkew:  0,
		DotCount: 1,
	}
	var driver base64Captcha.Driver
	driver = param.DriverDigit
	c := base64Captcha.NewCaptcha(driver, store)
	captchaId, captchaImg, err = c.Generate()
	if err != nil {
		return captchaId, captchaImg, err
	}
	CaptchaVal := c.Store.Get(captchaId, true)
	if _, err = redis.Set(captchaId, CaptchaVal, t).Result(); err != nil {
		return captchaId, captchaImg, err
	}
	return captchaId, captchaImg, nil
}

func VerifyCaptcha(redis *redis.Client, id, VerifyValue string) error {
	defer redis.Del(id)
	val, _ := redis.Get(id).Result()
	if val == "" {
		return errors.New("verify code expired")
	}
	if val != VerifyValue {
		return errors.New("verify code not right")
	}
	return nil
}
