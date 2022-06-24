package file

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver/v3"
)

func Unzip(src string, dest string) (filenames []string, err error) {

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

func UnArchiver(src string, dest string, suffix string) (filenames []string, err error) {
	err = archiver.Unarchive(src, dest)
	if err != nil {
		return
	}
	filenames, err = WalkDir(dest, suffix)
	if err != nil {
		return
	}
	return
}

func Archiver(src []string, dest string) (err error) {
	err = archiver.Archive(src, dest)
	//if err != nil {
	//	return
	//}
	//filenames, err = WalkDir(dest, suffix)
	//if err != nil {
	//	return
	//}
	return
}

func ListDir(dirPth string, sub string) (files []string, err error) {
	files = make([]string, 0)
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}
	PthSep := string(os.PathSeparator)
	sub = strings.ToUpper(sub) //忽略后缀匹配的大小写
	for _, fi := range dir {
		if fi.IsDir() { // 忽略目录
			continue
		}
		if strings.Contains(strings.ToUpper(fi.Name()), sub) { //匹配文件
			files = append(files, dirPth+PthSep+fi.Name())
		}
	}
	return files, nil
}

func ListDirFileInfo(dirPth string, sub string) (files []os.FileInfo, err error) {
	files = make([]os.FileInfo, 0)
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}
	sub = strings.ToUpper(sub) //忽略后缀匹配的大小写
	for _, fi := range dir {
		if fi.IsDir() { // 忽略目录
			continue
		}
		if strings.Contains(strings.ToUpper(fi.Name()), sub) { //匹配文件
			files = append(files, fi)
		}
	}
	return files, nil
}

func WalkDir(dirPth, sub string) (files []string, err error) {
	files = make([]string, 0)
	sub = strings.ToUpper(sub)                                                           //忽略后缀匹配的大小写
	err = filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error { //遍历目录
		if fi.IsDir() { // 忽略目录
			return nil
		}
		if strings.Contains(strings.ToUpper(fi.Name()), sub) {
			files = append(files, filename)
		}
		return nil
	})
	return files, err
}

func GetFileNameWithOutSuffix(fn string) string {
	full := path.Base(fn)
	fs := path.Ext(full)
	return strings.TrimSuffix(fn, fs)
}
