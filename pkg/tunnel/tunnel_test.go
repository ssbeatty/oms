package tunnel

import (
	"encoding/json"
	"github.com/ssbeatty/oms/pkg/transport"
	"io/ioutil"
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

var (
	host Host
	conf *transport.ClientConfig
)

func init() {
	data, err := ioutil.ReadFile("../transport/hosts")
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &host)
	if err != nil {
		return
	}
	conf = &transport.ClientConfig{
		Host:     host.Addr,
		User:     host.User,
		Password: host.PassWord,
		Port:     host.Port,
	}
}

func TestLocalTunnel(t *testing.T) {
	tunnel := NewSSHTunnel(
		conf, ":38080", "127.0.0.1:9000", LocalMode)

	go tunnel.Start()
	time.Sleep(time.Second)
	t.Log(tunnel.GetErrorMsg())

	select {}
}

func TestRemoteTunnel(t *testing.T) {
	tunnel := NewSSHTunnel(
		conf, "127.0.0.1:8082", ":8082", RemoteMode)

	go tunnel.Start()
}
