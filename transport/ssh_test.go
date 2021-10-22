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
var client *Client

func init() {
	data, err := ioutil.ReadFile("./hosts")
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &host)
	if err != nil {
		return
	}

	client, err = NewClient(host.Addr, host.Port, host.User, host.PassWord, nil)
	if err != nil {
		return
	}
}

func TestSampleCmd(t *testing.T) {

	session, err := client.NewSessionWithPty(20, 20)
	assert.Nil(t, err)

	result, err := session.Output("ls -al")
	assert.Nil(t, err)

	t.Log(string(result))
}

func TestLongTimeCmd(t *testing.T) {

	session, err := client.NewSessionWithPty(20, 20)
	assert.Nil(t, err)

	session.SetStdout(os.Stdout)
	session.SetStderr(os.Stdout)

	quitCh := make(chan bool)

	go func() {
		time.Sleep(5 * time.Second)
		quitCh <- true
	}()

	session.RunTaskWithQuit("sleep 60", quitCh)
}

func TestConnectionPing(t *testing.T) {
	_, err := client.NewSession()
	assert.Nil(t, err)

	client.NewSession()
	time.Sleep(5 * time.Second)
}
