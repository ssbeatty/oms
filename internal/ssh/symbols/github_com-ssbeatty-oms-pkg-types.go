// Code generated by 'yaegi extract github.com/ssbeatty/oms/pkg/types'. DO NOT EDIT.

package symbols

import (
	"github.com/ssbeatty/oms/pkg/transport"
	"github.com/ssbeatty/oms/pkg/types"
	"reflect"
)

func init() {
	Symbols["github.com/ssbeatty/oms/pkg/types/types"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"GetSchema":   reflect.ValueOf(types.GetSchema),
		"ParseCaches": reflect.ValueOf(types.ParseCaches),

		// type definitions
		"BaseStep": reflect.ValueOf((*types.BaseStep)(nil)),
		"Step":     reflect.ValueOf((*types.Step)(nil)),

		// interface wrapper definitions
		"_Step": reflect.ValueOf((*_github_com_ssbeatty_oms_pkg_types_Step)(nil)),
	}
}

// _github_com_ssbeatty_oms_pkg_types_Step is an interface wrapper for Step type
type _github_com_ssbeatty_oms_pkg_types_Step struct {
	IValue     interface{}
	WConfig    func() interface{}
	WCreate    func(conf []byte) (types.Step, error)
	WDesc      func() string
	WExec      func(session *transport.Session, sudo bool) ([]byte, error)
	WGetSchema func() (interface{}, error)
	WID        func() string
	WName      func() string
	WSetID     func(id string)
}

func (W _github_com_ssbeatty_oms_pkg_types_Step) Config() interface{} {
	return W.WConfig()
}
func (W _github_com_ssbeatty_oms_pkg_types_Step) Create(conf []byte) (types.Step, error) {
	return W.WCreate(conf)
}
func (W _github_com_ssbeatty_oms_pkg_types_Step) Desc() string {
	return W.WDesc()
}
func (W _github_com_ssbeatty_oms_pkg_types_Step) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	return W.WExec(session, sudo)
}
func (W _github_com_ssbeatty_oms_pkg_types_Step) GetSchema() (interface{}, error) {
	return W.WGetSchema()
}
func (W _github_com_ssbeatty_oms_pkg_types_Step) ID() string {
	return W.WID()
}
func (W _github_com_ssbeatty_oms_pkg_types_Step) Name() string {
	return W.WName()
}
func (W _github_com_ssbeatty_oms_pkg_types_Step) SetID(id string) {
	W.WSetID(id)
}
