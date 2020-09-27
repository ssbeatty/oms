package utils

import (
	"crypto/md5"
	"fmt"
	"io"
)

func NewMd5(s string)string{
	w := md5.New()
	io.WriteString(w, s)
	md5str2 := fmt.Sprintf("%x", w.Sum(nil))
	return md5str2
}

