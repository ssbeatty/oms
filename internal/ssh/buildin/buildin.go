package buildin

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/ssbeatty/oms/pkg/transport"
	"github.com/ssbeatty/oms/pkg/types"
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

	cfg *runCmdStepConfig
}

type runCmdStepConfig struct {
	Cmd string `json:"cmd" jsonschema:"required=true"`
}

func (bs *RunCmdStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	if sudo {
		return session.Sudo(bs.cfg.Cmd, session.Client.Conf.Password)
	}

	return session.Output(bs.cfg.Cmd)
}

func (bs *RunCmdStep) Create(conf []byte) (types.Step, error) {
	cfg := &runCmdStepConfig{}

	err := json.Unmarshal(conf, cfg)
	if err != nil {
		return nil, err
	}
	return &RunCmdStep{
		cfg: cfg,
	}, nil
}

func (bs *RunCmdStep) Config() interface{} {
	return bs.cfg
}

func (bs *RunCmdStep) Name() string {
	return StepNameCMD
}

func (bs *RunCmdStep) Desc() string {
	return "执行一条命令"
}

func (bs *RunCmdStep) GetSchema() (interface{}, error) {

	return types.GetSchema(bs.cfg)
}

// RunShellStep 执行shell
type RunShellStep struct {
	types.BaseStep

	cfg *runShellStepConfig
}

type runShellStepConfig struct {
	Shell string `json:"shell" jsonschema:"required=true"`
}

func (bs *RunShellStep) Exec(session *transport.Session, sudo bool) ([]byte, error) {
	tmpPath := filepath.ToSlash(filepath.Join(ShellTmpPath, uuid.NewString()))

	return session.RunScript(bs.cfg.Shell, sudo, tmpPath)
}

func (bs *RunShellStep) Create(conf []byte) (types.Step, error) {
	cfg := &runShellStepConfig{}

	err := json.Unmarshal(conf, cfg)
	if err != nil {
		return nil, err
	}
	return &RunShellStep{
		cfg: cfg,
	}, nil
}

func (bs *RunShellStep) Config() interface{} {
	return bs.cfg
}

func (bs *RunShellStep) Name() string {
	return StepNameShell
}

func (bs *RunShellStep) Desc() string {
	return "执行shell脚本"
}

func (bs *RunShellStep) GetSchema() (interface{}, error) {

	return types.GetSchema(bs.cfg)
}
