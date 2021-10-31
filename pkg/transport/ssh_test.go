package transport

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"io/ioutil"
	"oms/pkg/cache"
	"oms/pkg/utils"
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

func TestSudoCmd(t *testing.T) {

	session, err := client.NewSessionWithPty(20, 20)
	assert.Nil(t, err)

	result, err := session.Sudo("ls -l /root", host.PassWord)
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
	session, err := client.NewSession()
	assert.Nil(t, err)

	err = session.Run("cd /")
	assert.Nil(t, err)
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

	h, _ := client.sftpClient.Lstat("/bin")
	t.Log(h.Mode()&fs.ModeType == fs.ModeSymlink)

	s, _ := client.sftpClient.ReadLink("/bin")
	t.Log(s)

	t.Log(client.sftpClient.Getwd())

	infos, _ := client.ReadDir("/")
	for _, info := range infos {
		mode := info.Mode() & fs.ModeType
		if mode == fs.ModeSymlink {
			t.Log(client.RealPath(info.Name()))
		}
	}
}

func TestSSHStat(t *testing.T) {
	status := NewStatus()
	for i := 0; i < 10; i++ {
		GetAllStats(client.sshClient, status, nil)
		t.Log(status.CPU)
		time.Sleep(time.Second)
	}

	assert.NotNil(t, status)
}
