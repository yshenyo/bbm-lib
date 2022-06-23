package encryption

import (
	"fmt"
	"testing"
)

func TestEncode(t *testing.T) {
	var in = "teststring"
	out, err := Encode(in)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(out)
}

func TestDecode(t *testing.T) {
	var in = "f55725686b5f33121597a490d020f625"
	out, err := Decode(in)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(out)

}

func TestDecode22(t *testing.T) {
	var in = "fffffffffffff"
	out, err := Decode(in)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(out)

}

func TestEncodeWithCheck(t *testing.T) {
	var in = "2774a11f5ed6188e83734127fa195508"
	out, err := EncodeWithCheck(in)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(out)
}

func TestPasswordMd5(t *testing.T) {
	var in = "jhjdHHH34918"
	out := PasswordMd5(in)
	fmt.Println(out)
}
