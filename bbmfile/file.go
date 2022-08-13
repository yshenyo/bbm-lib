package bbmfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

type BBMFileObject struct {
	Path          string `json:"path"`
	LocalPath     string `json:"local_path"`
	Host          string `json:"host"`
	Authorization string `json:"authorization"`
	StoreFilePath string `json:"store_file_path"`
}

type Response struct {
	StatusCode int            `json:"status_code,omitempty"`
	MessageEN  string         `json:"message_en,omitempty"`
	MessageCN  string         `json:"message_cn,omitempty"`
	Detail     string         `json:"detail,omitempty"` //error detail information
	Alert      string         `json:"alert,omitempty"`
	ToWiki     string         `json:"to_wiki,omitempty"`
	Data       FileUploadResp `json:"data"`
}

type FileUploadResp struct {
	Path         string `json:"path"`
	DownloadUUID string `json:"download_uuid"`
}

func (file BBMFileObject) Put() (err error, path string, downloadUUID string) {
	client := &http.Client{}
	//构建server 对象
	values := map[string]io.Reader{
		"file":      mustOpen(file.LocalPath),
		"file_path": strings.NewReader(file.Path),
	}
	headers := map[string]string{
		"Authorization": file.Authorization,
		"FilePath":      file.StoreFilePath,
	}
	url := fmt.Sprintf("%s%s", file.Host, "/v1/file-upload")
	err, res := Upload(client, url, values, headers)
	if err != nil {
		return
	}
	return nil, res.Path, res.DownloadUUID
}

func Upload(client *http.Client, url string, values map[string]io.Reader, headers map[string]string) (err error, result FileUploadResp) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// 添加文件
		if x, ok := r.(*os.File); ok {
			if fw, err = w.CreateFormFile(key, x.Name()); err != nil {
				return fmt.Errorf("CreateFormFile error %v", err), result
			}
		} else { //添加字符串
			if fw, err = w.CreateFormField(key); err != nil {
				return fmt.Errorf("CreateFormField error %v", err), result
			}
		}
		if _, err = io.Copy(fw, r); err != nil {
			return
		}

	}
	w.Close()

	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return fmt.Errorf("request url %v error %v", url, err), result
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request url %v error %v", url, err), result
	}

	// 检查返回状态吗
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("request fail url %v error %v,bad status:%s", url, err, res.Status), result
	}
	rbody, _ := ioutil.ReadAll(res.Body)
	var resp Response
	_ = json.Unmarshal(rbody, &resp)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request fail url %v error %v,response %v", url, err, string(rbody)), result
	}
	return nil, resp.Data
}

func mustOpen(f string) *os.File {
	r, err := os.Open(f)
	if err != nil {
		panic(err)
	}
	return r
}
