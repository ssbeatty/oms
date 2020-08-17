package ssh

import (
	"fmt"
	"testing"
)

func TestNewWithAgent(t *testing.T) {

	c, err := NewWithAgent("118.190.117.250", "3009", "root")
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

func TestNewClient(t *testing.T) {
	c, err := NewClient("sasaba.net", "6666", "root", "wang199564", "")
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

func TestNewWithPrivateKey(t *testing.T) {
	c, err := NewWithPrivateKey("192.168.5.154", "22", "root", "123456")
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

func TestBash(t *testing.T) {
	c, err := NewClient("192.168.8.101", "22", "root", "199564", "")
	if err != nil {
		panic(err)
	}
	defer c.Close()

	err = c.Exec1("yum update")
	if err != nil {
		panic(err)
	}
}
