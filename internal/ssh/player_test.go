package ssh

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	r := RunCmdStep{}
	schema, err := r.GetSchema(&r)
	if err != nil {
		return
	}

	fmt.Println(string(schema))

	s := RunShellStep{}
	schema, err = s.GetSchema(&s)
	if err != nil {
		return
	}

	fmt.Println(string(schema))

	f := FileUploadStep{}
	schema, err = f.GetSchema(&f)
	if err != nil {
		return
	}

	fmt.Println(string(schema))
}

func TestPlayerRun(t *testing.T) {
	var steps []Step

	steps = append(steps, &RunCmdStep{
		Cmd: "ls -a",
		BaseStep: BaseStep{
			id: "run ls -a",
		},
	})

	steps = append(steps, &RunShellStep{
		Shell: "ls -l",
		BaseStep: BaseStep{
			id: "run ls -l",
		},
	})

	player := NewPlayer(client, steps)

	output, err := player.Run(context.Background())
	if err != nil {
		return
	}

	fmt.Println(string(output))
}
