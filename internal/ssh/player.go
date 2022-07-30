package ssh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/swaggest/jsonschema-go"
	"oms/pkg/transport"
	"reflect"
)

const (
	UploadPath = "upload"
	CMDName    = "--name"
	CMDScheme  = "--scheme"
	CMDClient  = "--client"
	CMDParams  = "--params"
)

type Step interface {
	Exec(session *transport.Session) ([]byte, error)
	GetSchema(instance Step) ([]byte, error)
	Create() Step
	Name() string
	ID() string
	SetID(id string)
	ParseCaches(instance Step) []string
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

func (p *Player) Run(ctx context.Context) ([]byte, error) {
	var (
		err     error
		buf     bytes.Buffer
		session *transport.Session
		quit    = make(chan struct{}, 1)
	)

	defer close(quit)

	go func() {
		select {
		case <-ctx.Done():
			if session != nil {
				session.Close()
			}
		case <-quit:
			return
		}
	}()

	for _, step := range p.Steps {
		session, err = p.client.NewPty()
		if err != nil {
			return buf.Bytes(), err
		}

		// todo 优化样式
		buf.WriteString(fmt.Sprintf("[Step %8s] ==> %s\n", step.Name(), step.ID()))
		msg, err := step.Exec(session)
		buf.Write(msg)

		if err != nil {
			return buf.Bytes(), err
		}

		session.Close()
	}

	return buf.Bytes(), nil
}

// build in

type BaseStep struct {
	id string // 任务步骤标识
}

func (bs *BaseStep) ParseCaches(instance Step) []string {
	var ret []string
	v := reflect.ValueOf(instance)

	t := reflect.TypeOf(instance).Elem()
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).Tag.Get("format") == "data-url" {
			ret = append(ret, v.Elem().FieldByName(t.Field(i).Name).String())
		}
	}
	return ret
}

func (bs *BaseStep) Exec(*transport.Session) ([]byte, error) {

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

func (bs *BaseStep) SetID(id string) {
	bs.id = id
}
