package matcher

import "regexp"

func MatchRegString(s string, reg string) (rl []string, err error) {
	rp, err := regexp.Compile(reg)
	if err != nil {
		return
	}
	rl = rp.FindAllString(s, -1)
	return
}
