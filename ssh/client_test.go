package ssh

import (
	"fmt"
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

}
