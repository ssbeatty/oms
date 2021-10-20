package transport

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

type Host struct {
	Addr     string `json:"addr"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	PassWord string `json:"password"`
	KeyBytes []byte `json:"key_bytes"`
}

var host Host

func init() {
	data, err := ioutil.ReadFile("./hosts")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &host)
	if err != nil {
		panic(err)
	}
}

func TestSampleCmd(t *testing.T) {
	client, err := NewClient(host.Addr, host.Port, host.User, host.PassWord, nil)
	assert.Nil(t, err)

	session, err := client.NewSessionWithPty(20, 20)
	assert.Nil(t, err)

	result, err := session.Output("ls -al")
	assert.Nil(t, err)

	t.Log(string(result))
}

func TestLongTimeCmd(t *testing.T) {
	client, err := NewClient(host.Addr, host.Port, host.User, host.PassWord, nil)
	assert.Nil(t, err)

	session, err := client.NewSessionWithPty(20, 20)
	assert.Nil(t, err)

	session.SetStdout(os.Stdout)
	session.SetStderr(os.Stdout)

	quitCh := make(chan bool)

	go func() {
		time.Sleep(5 * time.Second)
		quitCh <- true
	}()

	session.RunTaskWithQuit("sleep 300", quitCh)
}
