package utils

import "testing"

func TestNewMd5NewMd5(t *testing.T) {
	md5_str := NewMd5("123123")
	t.Log(md5_str)
}
