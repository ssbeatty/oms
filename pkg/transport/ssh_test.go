package transport

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/fs"
	"io/ioutil"
	"math/big"
	"net"
	"oms/pkg/cache"
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

func InetNtoA(ip int64) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

func InetAtoN(ip string) int64 {
	ret := big.NewInt(0)
	ret.SetBytes(net.ParseIP(ip).To4())
	return ret.Int64()
}

func (h *Host) serialize() int64 {
	return InetAtoN(h.Addr)
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
	session, err := client.NewSession()
	assert.Nil(t, err)

	err = session.Run("cd /")
	assert.Nil(t, err)
}

func TestConnCache(t *testing.T) {
	host1 := Host{Addr: "127.0.0.1"}
	host2 := Host{Addr: "127.0.0.2"}
	host3 := Host{Addr: "127.0.0.3"}

	assert.EqualValues(t, InetNtoA(host1.serialize()), "127.0.0.1")
	assert.EqualValues(t, InetNtoA(host2.serialize()), "127.0.0.2")
	assert.EqualValues(t, InetNtoA(host3.serialize()), "127.0.0.3")

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
