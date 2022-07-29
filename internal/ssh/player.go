package ssh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/swaggest/jsonschema-go"
	"oms/pkg/transport"
)

const (
	CMDName   = "--name"
	CMDScheme = "--scheme"
	CMDClient = "--client"
	CMDParams = "--params"
)

type Step interface {
	Exec(*transport.Client) ([]byte, error)
	GetSchema(instance Step) ([]byte, error)
	Create() Step
	Name() string
	ID() string
}

type Player struct {
	client *transport.Client
	Steps  []Step `json:"steps"`
}

func NewPlayer(client *transport.Client, steps []Step) *Player {
	return &Player{
		client: client,
		Steps:  steps,
	}
}

func (p *Player) Run() ([]byte, error) {
	var buf bytes.Buffer

	for _, step := range p.Steps {
		// todo 优化样式
		buf.WriteString(fmt.Sprintf("[Step %8s] ==> %s\n", step.Name(), step.ID()))
		msg, err := step.Exec(p.client)
		buf.Write(msg)

		if err != nil {
			return buf.Bytes(), err
		}
	}

	return buf.Bytes(), nil
}

// build in

type BaseStep struct {
	id string // 任务步骤标识
}

func (bs *BaseStep) Exec(*transport.Client) ([]byte, error) {

	return nil, nil
}

func (bs *BaseStep) GetSchema(instance Step) ([]byte, error) {
	reflector := jsonschema.Reflector{}

	schema, err := reflector.Reflect(instance)
	if err != nil {
		return nil, err
	}

	j, err := json.MarshalIndent(schema, "", " ")
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (bs *BaseStep) Create() Step {
	return nil
}

func (bs *BaseStep) Name() string {
	return ""
}

func (bs *BaseStep) ID() string {
	return bs.id
}
