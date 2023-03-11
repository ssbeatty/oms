package buildin

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/ssbeatty/oms/pkg/transport"
	"github.com/ssbeatty/oms/pkg/types"
	"os/exec"
	"path/filepath"
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

	// ShellTmpPath default in ~/.oms
	ShellTmpPath = ".oms"
)

// RunCmdStep 执行cmd
type RunCmdStep struct {
	types.BaseStep
	Cmd string `json:"cmd" jsonschema:"required=true"`
}

func (bs *RunCmdStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	if sudo {
		return session.Sudo(bs.Cmd, session.Client.Conf.Password)
	}

	return session.Output(bs.Cmd)
}

func (bs *RunCmdStep) Create() types.Step {
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
	types.BaseStep
	Shell string `json:"shell" jsonschema:"required=true"`
}

func (bs *RunShellStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	tmpPath := filepath.ToSlash(filepath.Join(ShellTmpPath, uuid.NewString()))

	return session.RunScript(bs.Shell, sudo, tmpPath)
}

func (bs *RunShellStep) Create() types.Step {
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
	types.BaseStep
	Data       interface{}
	ScriptPath string
	name       string
	desc       string
	schema     interface{}
}

func (bs *PluginStep) Create() types.Step {
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

func (bs *PluginStep) GetSchema(instance types.Step) (interface{}, error) {
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
