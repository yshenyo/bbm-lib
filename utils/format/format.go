package format

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/spf13/cast"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// 	Package reflect implements run-time reflection,
//  allowing a program to manipulate objects with arbitrary types.
//  it is too slow and vulnerable, therefore if not necessary, don't use it.

//func StringsFormat(text string, data map[string]string) string {
//	keys := reflect.ValueOf(data).MapKeys()
//	for _, v := range keys {
//		key := v.String()
//		old := fmt.Sprintf("{%s}", key)
//		new := data[key]
//		text = strings.Replace(text, old, new, -1)
//	}
//	return text
//}

// StringsFormat format string with map
func StringsFormat(text string, data map[string]string) string {
	for k, v := range data {
		old := fmt.Sprintf("{%s}", k)
		n := v
		text = strings.Replace(text, old, n, -1)
	}
	return text
}

func StringFormatWitchViews(text string, data map[string]string) string {
	for k, v := range data {
		if k == "{use_global_view}" {
			text, _ = formatViews(text, v)
		} else {
			old := fmt.Sprintf("{%s}", k)
			n := v
			text = strings.Replace(text, old, n, -1)
		}
	}
	return text

}

func formatViews(sql string, ugv string) (ns string, err error) {
	if ugv == "false" {
		sql = strings.Replace(sql, "{{use_global_view}}", "v", -1)
		reg, err := regexp.Compile("(?U)######.+######")
		if err != nil {
			return ns, err
		}
		ns = reg.ReplaceAllString(sql, "")
	} else {
		sql = strings.Replace(sql, "{{use_global_view}}", "gv", -1)
		ns = strings.Replace(sql, "######", "", -1)
	}
	return ns, nil
}

func EncodingFormat(data []byte) (s string, err error) {
	c := GetStrCoding(data)
	if c == "GBK" {
		d, err := simplifiedchinese.GBK.NewEncoder().Bytes(data)
		if err != nil {
			return "", err
		}
		s = string(d)
		return s, nil
	}
	s = string(data)
	return
}

func isGBK(data []byte) bool {
	length := len(data)
	var i = 0
	for i < length {
		if data[i] <= 0x7f {
			i++
			continue
		} else {
			if data[i] >= 0x81 &&
				data[i] <= 0xfe &&
				data[i+1] >= 0x40 &&
				data[i+1] <= 0xfe &&
				data[i+1] != 0xf7 {
				i += 2
				continue
			} else {
				return false
			}
		}
	}
	return true
}

func preNUm(data byte) int {
	var mask byte = 0x80
	var num = 0
	for i := 0; i < 8; i++ {
		if (data & mask) == mask {
			num++
			mask = mask >> 1
		} else {
			break
		}
	}
	return num
}
func isUtf8(data []byte) bool {
	i := 0
	for i < len(data) {
		if (data[i] & 0x80) == 0x00 {
			i++
			continue
		} else if num := preNUm(data[i]); num > 2 {
			i++
			for j := 0; j < num-1; j++ {
				if (data[i] & 0xc0) != 0x80 {
					return false
				}
				i++
			}
		} else {
			return false
		}
	}
	return true
}

func GetStrCoding(data []byte) string {
	if isUtf8(data) == true {
		return "UTF8"
	} else if isGBK(data) == true {
		return "GBK"
	} else {
		return "UNKNOWN"
	}
}

func IntSliceJoin(s []int, sub string) string {
	sl := make([]string, 0)
	for _, n := range s {
		sl = append(sl, cast.ToString(n))
	}
	return strings.Join(sl, sub)
}

func Divide(a, b float64) float64 {
	if b == 0 {
		return a
	}
	return math.Trunc(a/b*1e2+0.5) * 1e-2
}

func StringToIntArray(s string, sep string) []int {
	result := make([]int, 0)
	if sep == "" {
		sep = ","
	}
	for _, v := range strings.Split(s, sep) {
		n := cast.ToInt(v)
		if n == 0 {
			continue
		}
		result = append(result, n)
	}
	return result
}

func MD5String(str string) (id string) {
	h := md5.New()
	defer func() {
		if err := recover(); err != nil {
			h.Write([]byte(str))
			id = hex.EncodeToString(h.Sum(nil))
		}
	}()
	h.Write([]byte(str))
	id = hex.EncodeToString(h.Sum(nil))
	return
}
