package ssh

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"oms/internal/ssh/buildin"
	"oms/pkg/transport"
	"testing"
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
)

func init() {
	data, err := ioutil.ReadFile("../../pkg/transport/hosts")
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &host)
	if err != nil {
		return
	}

	client, err = transport.New(host.Addr, host.User, host.PassWord, "", nil, host.Port)
	if err != nil {
		return
	}
}

func TestGenJsonSchema(t *testing.T) {
	r := buildin.RunCmdStep{}
	schema, err := r.GetSchema(&r)
	if err != nil {
		return
	}

	marshal, _ := json.Marshal(schema)

	fmt.Println(string(marshal))

	s := buildin.RunShellStep{}
	schema, err = s.GetSchema(&s)
	if err != nil {
		return
	}
	marshal, _ = json.Marshal(schema)

	fmt.Println(string(marshal))

	f := buildin.FileUploadStep{}
	schema, err = f.GetSchema(&f)
	if err != nil {
		return
	}

	marshal, _ = json.Marshal(schema)

	fmt.Println(string(marshal))
}

func TestPlayerRun(t *testing.T) {
	var steps []buildin.Step

	steps = append(steps, &buildin.RunCmdStep{
		Cmd: "ls -a",
	})

	steps = append(steps, &buildin.RunShellStep{
		Shell: "ls -l",
	})

	player := NewPlayer(client, steps, true)

	output, err := player.Run(context.Background())
	if err != nil {
		return
	}

	fmt.Println(string(output))
}
