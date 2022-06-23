package notify

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"bbm_lib/utils/encryption"
	"gopkg.in/gomail.v2"
)

type notify struct{}

type NotifyInterface interface {
	SendEmailMsg(m *gomail.Message, emailConfig EmailConfig) error
	SendWeChatMsg(content, receiver string, weChatConfig WechatConfig) error
	SendDingTalkMsg(content string, dingTalkConfig DingTalkConfig) error
	SendScriptMsg(content, receiver string, scriptConfig ScriptConfig) error
}

func NewNotify() NotifyInterface {
	return &notify{}
}

type EmailConfig struct {
	SmtpSmarthost    string `json:"smtp_smarthost"`
	SmtpFrom         string `json:"smtp_from"`
	SmtpAuthUsername string `json:"smtp_auth_username"`
	SmtpAuthPassword string `json:"smtp_auth_password"`
	SmtpRequireTls   bool   `json:"smtp_require_tls"`
	SendTo           string `json:"send_to"`
}

type WechatConfig struct {
	CorpId     string `json:"corp_id"`
	CorpSecret string `json:"corp_secret"`
	AgentId    int    `json:"agent_id"`
}

type DingTalkConfig struct {
	DingTalkWebhook string `json:"webhook"`
}

type ScriptConfig struct {
	Script         string `json:"script"`
	ScriptReceiver string `json:"script_receiver"`
}

func (n notify) SendEmailMsg(m *gomail.Message, emailConfig EmailConfig) error {
	p, err := encryption.Decode(emailConfig.SmtpAuthPassword)
	if err == nil && p != "" {
		emailConfig.SmtpAuthPassword = p
	}
	m.SetHeader("From", emailConfig.SmtpFrom)
	hp := strings.Split(emailConfig.SmtpSmarthost, ":")
	var port int = 465
	if len(hp) == 2 {
		port, _ = strconv.Atoi(hp[1])
	}
	d := gomail.NewDialer(hp[0], port, emailConfig.SmtpAuthUsername, emailConfig.SmtpAuthPassword)
	if emailConfig.SmtpRequireTls == true {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if err = d.DialAndSend(m); err != nil {
		return fmt.Errorf("send Email error %v", err)
	}
	return nil
}

func (n notify) SendWeChatMsg(content, receiver string, wechatConfgig WechatConfig) error {
	p, err := encryption.Decode(wechatConfgig.CorpSecret)
	if err == nil && p != "" {
		wechatConfgig.CorpSecret = p
	}
	token, err := getWechatToken(wechatConfgig.CorpId, wechatConfgig.CorpSecret)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	data := map[string]interface{}{
		"touser":  receiver,
		"agentid": wechatConfgig.AgentId,
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}
	req, _ := json.Marshal(data)
	res, err := http.Post(url, "application/json;charset=utf-8", bytes.NewBuffer(req))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return err
}

func (n notify) SendDingTalkMsg(content string, dingTalkConfig DingTalkConfig) error {
	data := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}
	req, _ := json.Marshal(data)
	res, err := http.Post(dingTalkConfig.DingTalkWebhook, "application/json;charset=utf-8", bytes.NewBuffer(req))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func (n notify) SendScriptMsg(content, receiver string, scriptConfig ScriptConfig) error {
	cmd := exec.Command(scriptConfig.Script, receiver, content)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("exec cmd '%v' error  %v", cmd, err)
	}
	return nil
}

func getWechatToken(id string, secret string) (token string, err error) {
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", id, secret)
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	type wechatBody struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`

		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}

	var wb wechatBody
	err = json.Unmarshal(body, &wb)
	if wb.Errmsg != "ok" {
		return "", errors.New(wb.Errmsg)
	}

	return wb.AccessToken, err
}
