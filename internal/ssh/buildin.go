package ssh

import (
	"encoding/json"
	"oms/internal/utils"
	"oms/pkg/transport"
	"os/exec"
)

const (
	StepNameCMD   = "cmd"
	StepNameShell = "shell"
	StepNameFile  = "file"
)

// RunCmdStep 执行cmd
type RunCmdStep struct {
	BaseStep
	Cmd string `json:"cmd" required:"true"`
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

// RunShellStep 执行shell
type RunShellStep struct {
	BaseStep
	Shell string `json:"shell" required:"true"`
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

// FileUploadStep 上传文件
type FileUploadStep struct {
	BaseStep
	File   string `json:"file" required:"true" format:"data-url"`
	Remote string `json:"remote" required:"true"`
}

func (bs *FileUploadStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	err := session.Client.NewSftpClient()
	if err != nil {
		return nil, err
	}
	if exists, err := utils.PathExists(bs.File); !exists {
		return nil, err
	}

	if sudo {
		// todo change fs perm
	}

	return nil, session.Client.UploadFile(bs.File, bs.Remote)
}

func (bs *FileUploadStep) Create() Step {
	return &FileUploadStep{}
}

func (bs *FileUploadStep) Name() string {
	return StepNameFile
}

// PluginStep 调用外部程序比如python或者go的脚本来获取输出
// 参数为连接的信息和插件模块的参数
// 插件需要提供Scheme 以供渲染表单
type PluginStep struct {
	BaseStep
	Data       interface{} `json:"data"`
	ScriptPath string
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
	name, err := exec.Command(bs.ScriptPath, CMDName).Output()
	if err != nil {
		return ""
	}
	return string(name)
}

func (bs *PluginStep) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &bs.Data)
	if err != nil {
		return err
	}
	return nil
}

func (bs *PluginStep) GetSchema(instance Step) (interface{}, error) {
	ret := make(map[string]interface{})

	_, err := exec.LookPath(bs.ScriptPath)
	if err != nil {
		return nil, err
	}

	b, err := exec.Command(bs.ScriptPath, CMDScheme).Output()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (bs *PluginStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	configJson, _ := json.Marshal(session.Client.Conf)
	params, _ := json.Marshal(bs.Data)
	return exec.Command(bs.ScriptPath, CMDClient, string(configJson), CMDParams, string(params)).Output()
}
