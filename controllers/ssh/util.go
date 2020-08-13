package ssh

import (
	"bufio"
	"crypto"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

func KeyFile() string {

	home, err := homedir.Dir()
	if err != nil {
		println(err.Error())
		return ""
	}
	key := filepath.ToSlash(path.Join(home, ".ssh/id_rsa"))
	log.Println(key)
	return key
}
func FileExist(file string) bool {
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
func MkdirAll(path string) error {
	// 检测文件夹是否存在   若不存在  创建文件夹
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(path, os.ModePerm)
		}
	}
	return nil
}

//Md5File 计算md5
func Md5File(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	hash := crypto.MD5.New()
	_, err = io.Copy(hash, r)
	if err != nil {
		return "", err
	}

	out := hex.EncodeToString(hash.Sum(nil))
	return out, nil
}
