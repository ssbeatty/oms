package ssh

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient("192.168.8.101", 22, "root", "123456", "")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()
	b, err := c.Output("id")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(b))
}

func TestReadDir(t *testing.T) {
	keyPath := "C:\\Users\\Administrator\\.ssh\\id_rsa"
	keyText, err := ioutil.ReadFile(keyPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	c, err := NewClient("192.168.1.118", 22, "pi", "123456", string(keyText))
	if err != nil {
		fmt.Println(err)
		return
	}
	info, err := c.ReadDir("/home/pi")
	fmt.Println(info)

}
