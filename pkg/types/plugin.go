package types

import (
	"github.com/ssbeatty/jsonschema"
	"github.com/ssbeatty/oms/pkg/transport"
	"reflect"
	"strings"
)

type Step interface {
	Exec(session *transport.Session, sudo bool) ([]byte, error)
	GetSchema(instance Step) (interface{}, error)
	Create() Step
	Name() string
	Desc() string
	ID() string
	SetID(id string)
	ParseCaches(instance Step) []string
}

// build in

type BaseStep struct {
	id string // 任务步骤标识
}

func readStringArray(v reflect.Value) (vals []string) {
	count := v.Len()

	for i := 0; i < count; i++ {
		child := v.Index(i)
		s := child.String()
		vals = append(vals, s)
	}

	return
}

func (bs *BaseStep) ParseCaches(instance Step) []string {
	var ret []string
	v := reflect.ValueOf(instance)

	t := reflect.TypeOf(instance).Elem()
	for i := 0; i < t.NumField(); i++ {
		if strings.Contains(t.Field(i).Tag.Get("jsonschema"), "format=data-url") {
			if t.Field(i).Type.Kind() == reflect.String {
				ret = append(ret, v.Elem().Field(i).String())
			} else if t.Field(i).Type.Kind() == reflect.Slice {
				ret = readStringArray(v.Elem().Field(i))
			}
		}
	}
	return ret
}

func (bs *BaseStep) Exec(*transport.Session) ([]byte, error) {

	return nil, nil
}

func (bs *BaseStep) GetSchema(instance Step) (interface{}, error) {
	ref := jsonschema.Reflector{DoNotReference: true}

	return ref.Reflect(instance), nil
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

func (bs *BaseStep) Desc() string {
	return ""
}

func (bs *BaseStep) SetID(id string) {
	bs.id = id
}
