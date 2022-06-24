package matcher

import (
	"bytes"
	"fmt"

	"github.com/benhoyt/goawk/interp"
)

func MatchAwkString(s string, exp string) (r string, err error) {
	input := bytes.NewReader([]byte(s))
	buf := new(bytes.Buffer)
	err = interp.Exec(exp, " ", input, buf)
	if err != nil {
		return
	}
	r = buf.String()
	return
}

func AwkMatcher() {
	buf := new(bytes.Buffer)
	input := bytes.NewReader([]byte("foo bar\n\nbaz buz"))
	err := interp.Exec("$0 { print $1 }", " ", input, buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("1aaaaaaa")
	fmt.Println(buf)
}
