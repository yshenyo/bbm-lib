package file

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// 判断所给路径是否为文件
func IsFile(path string) bool {
	return !IsDir(path)
}

// aaa/bbmgr/dist/200.html
// return 200.html
func GetFileName(filePath string) string {
	return path.Base(filePath)
}

// aaa/bbmgr/dist/200.html
// return aaa/bbmgr/dist/  ,  200.html
func GetFilePathSplit(filePath string) (basePath string, fileName string) {
	basePath, fileName = filepath.Split(filePath)
	return
}

func GetFileSuffix(fileName string) (fileSuffix string) {
	fileSuffix = path.Ext(fileName)
	return
}

func ReadFile(filePath string) (content string, err error) {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func WriteFile(filePath string, content string) (err error) {
	data := []byte(content)
	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
