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

var host Host
var client *transport.Client

func init() {
	data, err := ioutil.ReadFile("../transport/hosts")
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &host)
	if err != nil {
		return
	}

	client, err = transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, nil)
	if err != nil {
		return
	}
}

func TestLocalTunnel(t *testing.T) {
	tunnel := NewSSHTunnel(client.GetSSHClient(), ":8085", "127.0.0.1:15672", LocalMode)

	go tunnel.Start()

	time.Sleep(time.Second)
	t.Log(tunnel.GetErrorMsg())

}

func TestRemoteTunnel(t *testing.T) {
	tunnel := NewSSHTunnel(client.GetSSHClient(), "127.0.0.1:8082", ":8082", RemoteMode)

	go tunnel.Start()
}
