package transport_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"io/ioutil"
	"oms/internal/ssh"
	"oms/internal/utils"
	"oms/pkg/cache"
	"oms/pkg/transport"
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

func (h *Host) serialize() int64 {
	return utils.InetAtoN(h.Addr, h.Port)
}

var host Host
var client *transport.Client

func init() {
	data, err := ioutil.ReadFile("./hosts")
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &host)
	if err != nil {
		return
	}

	conf := transport.ClientConfig{
		Host:     host.Addr,
		User:     host.User,
		Password: host.PassWord,
		Port:     host.Port,
	}

	client, err = transport.New(&conf)
	if err != nil {
		return
	}

	err = client.NewSftpClient()
	if err != nil {
		return
	}
}

func TestSampleCmd(t *testing.T) {

	session, err := client.NewPty()
	assert.Nil(t, err)

	result, err := session.Output("ls")
	assert.Nil(t, err)

	fmt.Println(string(result))
}

func TestSampleInterCmd(t *testing.T) {

	session, err := client.NewPty()
	assert.Nil(t, err)

	result, err := client.OutputInteractively(session, "ls")
	assert.Nil(t, err)

	fmt.Println(string(result))
}

func TestSudoCmd(t *testing.T) {

	session, err := client.NewPty()
	assert.Nil(t, err)

	result, err := session.Sudo("sleep 60", host.PassWord)
	assert.Nil(t, err)
	t.Log(string(result))

	session, err = client.NewPty()
	assert.Nil(t, err)

	result, err = session.Sudo("ls -l", host.PassWord)
	assert.Nil(t, err)

	t.Log(string(result))
}

func TestLongTimeCmd(t *testing.T) {
	quitCh := make(chan bool)

	go func() {
		time.Sleep(5 * time.Second)
		quitCh <- true
	}()

	ssh.RunTaskWithQuit(client, "sleep 60", quitCh, os.Stdout)
}

func TestConnCache(t *testing.T) {
	host1 := Host{Addr: "127.0.0.1", Port: 22}
	host2 := Host{Addr: "127.0.0.2", Port: 22}
	host3 := Host{Addr: "127.0.0.3", Port: 22}

	assert.EqualValues(t, utils.InetNtoA(host1.serialize()), "127.0.0.1:22")
	assert.EqualValues(t, utils.InetNtoA(host2.serialize()), "127.0.0.2:22")
	assert.EqualValues(t, utils.InetNtoA(host3.serialize()), "127.0.0.3:22")

	l := cache.NewCache(2)

	l.Add(host1.serialize(), &host1)
	l.Add(host2.serialize(), &host2)
	l.Add(host3.serialize(), &host3)

	ret, _ := l.Get(host1.serialize())
	assert.Nil(t, ret)

	ret, _ = l.Get(host2.serialize())
	assert.NotNil(t, ret)
}

func TestNewClientWithSftp(t *testing.T) {
	err := client.NewSftpClient()
	assert.Nil(t, err)

	h, _ := client.GetSftpClient().Lstat("/bin")
	t.Log(h.Mode()&fs.ModeType == fs.ModeSymlink)

	s, _ := client.GetSftpClient().ReadLink("/bin")
	t.Log(s)

	t.Log(client.GetSftpClient().Getwd())

	infos, _ := client.ReadDir("/")
	for _, info := range infos {
		mode := info.Mode() & fs.ModeType
		if mode == fs.ModeSymlink {
			t.Log(client.RealPath(info.Name()))
		}
	}
}

func TestSSHStat(t *testing.T) {
	status := transport.NewStatus()
	for i := 0; i < 10; i++ {
		transport.GetAllStats(client, status, nil)
		t.Log(status.CPU)
		time.Sleep(time.Second)
	}

	assert.NotNil(t, status)
}

func TestRunScript(t *testing.T) {
	shell := `#/bin/bash
ls -a
echo 111
ls
echo 222
ls -lh
`
	session, err := client.NewPty()
	assert.Nil(t, err)

	output, err := session.RunScript(shell, true)
	assert.Nil(t, err)

	fmt.Println(string(output))
}

func TestChmod(t *testing.T) {
	err := client.Chmod("/root/test")

	assert.Nil(t, err)

}

func TestConnectionPing(t *testing.T) {
	ok, msg, err := client.SendRequest("keepalive@golang.org", true, nil)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(ok, msg)
}
