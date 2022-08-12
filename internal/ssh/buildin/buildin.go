package buildin

import (
	"encoding/json"
	"github.com/ssbeatty/jsonschema"
	"io/fs"
	"oms/pkg/transport"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	StepNameCMD       = "cmd"
	StepNameShell     = "shell"
	StepNameFile      = "file"
	StepMultiNameFile = "multi_file"
	StepNameZipFile   = "zip"
	StepNameYamlJson  = "json_yaml"

	GUIDLength  = 36
	CMDName     = "name"
	CMDDesc     = "desc"
	CMDScheme   = "schema"
	CMDCommand  = "exec"
	ParamClient = "--client"
	ParamParams = "--params"

	GOOSWindows   = "windows"
	DefaultFileFs = fs.FileMode(0644)
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

// RunCmdStep 执行cmd
type RunCmdStep struct {
	BaseStep
	Cmd string `json:"cmd" jsonschema:"required=true"`
}

func (bs *RunCmdStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	if sudo {
		return session.Sudo(bs.Cmd, session.Client.Conf.Password)
	}

	return session.Output(bs.Cmd)
}

func (bs *RunCmdStep) Create() Step {
	return &RunCmdStep{}
}

func (bs *RunCmdStep) Name() string {
	return StepNameCMD
}

func (bs *RunCmdStep) Desc() string {
	return "执行一条命令"
}

// RunShellStep 执行shell
type RunShellStep struct {
	BaseStep
	Shell string `json:"shell" jsonschema:"required=true"`
}

func (bs *RunShellStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {

	return session.RunScript(bs.Shell, sudo)
}

func (bs *RunShellStep) Create() Step {
	return &RunShellStep{}
}

func (bs *RunShellStep) Name() string {
	return StepNameShell
}

func (bs *RunShellStep) Desc() string {
	return "执行shell脚本"
}

// PluginStep 调用外部程序比如python或者go的脚本来获取输出
// 参数为连接的信息和插件模块的参数
// 插件需要提供Scheme 以供渲染表单
type PluginStep struct {
	BaseStep
	Data       interface{}
	ScriptPath string
	name       string
	desc       string
	schema     interface{}
}

func (bs *PluginStep) Create() Step {
	if bs.ScriptPath == "" {
		return nil
	}
	return &PluginStep{
		ScriptPath: bs.ScriptPath,
	}
}

func (bs *PluginStep) Name() string {
	if bs.name != "" {
		return bs.name
	}
	name, err := exec.Command(bs.ScriptPath, CMDName).CombinedOutput()
	if err != nil {
		return ""
	}

	bs.name = string(name)
	return string(name)
}

func (bs *PluginStep) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &bs.Data)
	if err != nil {
		return err
	}
	return nil
}

func (bs *PluginStep) MarshalJSON() ([]byte, error) {
	return json.Marshal(&bs.Data)
}

func (bs *PluginStep) GetSchema(instance Step) (interface{}, error) {
	if bs.schema != nil {
		return bs.schema, nil
	}
	ret := make(map[string]interface{})

	_, err := exec.LookPath(bs.ScriptPath)
	if err != nil {
		return nil, err
	}

	b, err := exec.Command(bs.ScriptPath, CMDScheme).CombinedOutput()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	bs.schema = ret

	return ret, nil
}

func (bs *PluginStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	configJson, _ := json.Marshal(session.Client.Conf)
	params, _ := json.Marshal(bs.Data)

	abs, err := filepath.Abs(bs.ScriptPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(
		abs, CMDCommand, ParamClient, string(configJson), ParamParams, string(params))
	cmd.Dir = filepath.Dir(bs.ScriptPath)
	return cmd.CombinedOutput()
}

func (bs *PluginStep) Desc() string {
	if bs.desc != "" {
		return bs.desc
	}
	desc, err := exec.Command(bs.ScriptPath, CMDDesc).CombinedOutput()
	if err != nil {
		return ""
	}
	bs.desc = string(desc)
	return string(desc)
}
