package types

import (
	"github.com/ssbeatty/jsonschema"
	"github.com/ssbeatty/oms/pkg/transport"
	"reflect"
	"strings"
)

type Step interface {
	ID() string
	SetID(id string)
	Exec(session *transport.Session, sudo bool) ([]byte, error)
	GetSchema() (interface{}, error)
	Create(conf []byte) (Step, error)
	Name() string
	Desc() string
	Config() interface{}
}

// build in

type BaseStep struct {
	id string // 任务步骤标识
}

func (bs *BaseStep) SetID(id string) {
	bs.id = id
}

func (bs *BaseStep) ID() string {
	return bs.id
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

func ParseCaches(conf interface{}) []string {
	var ret []string
	if conf == nil {
		return ret
	}

	v := reflect.ValueOf(conf)

	t := reflect.TypeOf(conf).Elem()
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

func GetSchema(config interface{}) (interface{}, error) {
	ref := jsonschema.Reflector{DoNotReference: true}

	return ref.Reflect(config), nil
}
