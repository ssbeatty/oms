package ssh

import (
	"encoding/json"
	"github.com/swaggest/jsonschema-go"
	"oms/pkg/transport"
	"os/exec"
)

const (
	StepNameCMD   = "cmd"
	StepNameShell = "shell"
	StepNameFile  = "file"
)

type Step interface {
	Exec(*transport.Client) ([]byte, error)
	GetSchema(instance Step) ([]byte, error)
	Create() Step
	Name() string
}

type Player struct {
	client *transport.Client
	Steps  []Step `json:"steps"`
}

// build in

type BaseStep struct {
	name string
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

// RunCmdStep 执行cmd
type RunCmdStep struct {
	BaseStep
	Cmd string `json:"cmd" required:"true"`
}

func (bs *RunCmdStep) Exec(client *transport.Client) ([]byte, error) {
	session, err := client.NewPty()
	if err != nil {
		return nil, err
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

func (bs *RunShellStep) Exec(client *transport.Client) ([]byte, error) {

	return client.RunScript(bs.Shell)
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

func (bs *FileUploadStep) Exec(client *transport.Client) ([]byte, error) {
	err := client.NewSftpClient()
	if err != nil {
		return nil, err
	}

	return nil, client.UploadFile(bs.File, bs.Remote)
}

func (bs *FileUploadStep) Create() Step {
	return &FileUploadStep{}
}

func (bs *FileUploadStep) Name() string {
	return StepNameFile
}

type PluginUploadStep struct {
	Data       interface{} `json:"data"`
	ScriptPath string
}

func (bs *PluginUploadStep) Create() Step {
	return &PluginUploadStep{}
}

func (bs *PluginUploadStep) Name() string {
	name, err := exec.Command(bs.ScriptPath, "--name").Output()
	if err != nil {
		return ""
	}
	return string(name)
}

func (bs *PluginUploadStep) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &bs.Data)
	if err != nil {
		return err
	}
	return nil
}

func (bs *PluginUploadStep) GetSchema(instance Step) ([]byte, error) {
	_, err := exec.LookPath(bs.ScriptPath)
	if err != nil {
		return nil, err
	}

	return exec.Command(bs.ScriptPath, "--scheme").Output()
}

func (bs *PluginUploadStep) Exec(client *transport.Client) ([]byte, error) {
	// todo
	return exec.Command(bs.ScriptPath, "--client").Output()
}
