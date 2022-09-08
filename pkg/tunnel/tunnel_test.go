package tunnel

import (
	"encoding/json"
	"io/ioutil"
	"oms/pkg/transport"
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
	host   Host
	client *transport.Client
	status = make(chan *transport.Client)
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

	client.Notify(status)
}

func TestLocalTunnel(t *testing.T) {
	tunnel := NewSSHTunnel(
		client.GetSSHClient(), ":38080", "127.0.0.1:9000", LocalMode)

	go tunnel.Start()
	time.Sleep(time.Second)
	t.Log(tunnel.GetErrorMsg())

	<-status
}

func TestRemoteTunnel(t *testing.T) {
	tunnel := NewSSHTunnel(
		client.GetSSHClient(), "127.0.0.1:8082", ":8082", RemoteMode)

	go tunnel.Start()
}
