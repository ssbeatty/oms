package ssh

import (
	"fmt"
	"testing"
)

func TestNewClient(t *testing.T) {
	c, err := NewClient("sasaba.net", 6666, "root", "wang199564", "")
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
