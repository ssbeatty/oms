package ssh

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/ssbeatty/oms/internal/config"
	"github.com/ssbeatty/oms/internal/models"
	"github.com/ssbeatty/oms/internal/ssh/buildin"
	"github.com/ssbeatty/oms/internal/ssh/symbols"
	"github.com/ssbeatty/oms/pkg/cache"
	"github.com/ssbeatty/oms/pkg/logger"
	"github.com/ssbeatty/oms/pkg/transport"
	"github.com/ssbeatty/oms/pkg/types"
	"github.com/ssbeatty/oms/pkg/utils"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	CMDTypeShell  = "cmd"
	CMDTypePlayer = "player"

	PollSize         = 2000
	manifestFilename = "manifest.yaml"
)

type Schema struct {
	Type   string      `json:"type"`
	Schema interface{} `json:"schema"`
	Desc   string      `json:"desc"`
}

type Command struct {
	Type       string
	Params     string
	Sudo       bool
	WindowSize WindowSize
}

type WindowSize struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

type Result struct {
	Seq      int    `json:"seq"`
	Status   bool   `json:"status"`
	HostId   int    `json:"host_id"`
	HostName string `json:"hostname"`
	Msg      string `json:"msg"`
	Addr     string `json:"addr"`
}

type Manager struct {
	fileList       *utils.SafeMap
	sshPoll        *cache.Cache
	logger         *logger.Logger
	notify         chan bool
	subClients     sync.Map
	supportPlugins map[string]types.Step // 注册所有的插件类型通过接口返回 表单渲染的格式
	cfg            *config.Conf
	statusChan     chan *transport.Client
	pluginPath     string
	interpreter    *interp.Interpreter
}

func NewManager(cfg *config.Conf) *Manager {
	pluginPath := filepath.Join(cfg.App.DataPath, config.PluginPath)

	return &Manager{
		notify:     make(chan bool, 10),
		subClients: sync.Map{},
		fileList:   utils.NewSafeMap(),
		sshPoll:    cache.NewCache(PollSize),
		logger:     logger.NewLogger("sshManager"),
		cfg:        cfg,
		statusChan: make(chan *transport.Client),
		pluginPath: pluginPath,
	}
}

// 从ssh manager poll删除死掉的客户端
func (m *Manager) doResolveStatus() {
	for {
		select {
		case c := <-m.statusChan:
			m.logger.Debugf("get client close status: addr: %s", c.Conf.Host)
			m.removeCache(c.Conf.Serialize())
			c.Close()

			if c.Conf.ID > 0 {
				_ = models.UpdateHostStatus(&models.Host{Id: c.Conf.ID, Status: false})
			}

			c = nil
		}
	}
}

func (m *Manager) Init() *Manager {
	go m.doNotifyFileTaskList()
	go m.doResolveStatus()

	m.initAllPlugins()

	return m
}

func (m *Manager) ReloadAllFilePlugins(pluginPath string) {
	_ = filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
		if info.Name() != manifestFilename {
			return nil
		}
		manifest, err := readManifest(path)
		if err != nil {
			m.logger.Errorf("Error when load manifest %s, error: %v", path, err)
			return nil
		}

		if _, ok := m.supportPlugins[manifest.DisplayName]; ok {
			// skip
			return nil
		}

		step, err := m.checkPlugin(manifest)
		if err != nil {
			m.logger.Errorf("Error when load plugin %s, error: %v", path, err)
			// continue
			return nil
		}

		m.supportPlugins[step.Name()] = step
		return nil
	})
}

// initAllPlugins 启动时加载所有插件
func (m *Manager) initAllPlugins() {
	if exists, _ := utils.PathExists(m.pluginPath); !exists {
		_ = os.MkdirAll(m.pluginPath, fs.ModePerm)
	}

	// set interpreter envs
	// PLUGIN_PATH current plugin path
	envSlice := os.Environ()
	envSlice = append(envSlice, fmt.Sprintf("PLUGIN_PATH=%s", m.pluginPath))

	i := interp.New(interp.Options{
		GoPath: m.pluginPath,
		Env:    envSlice,
	})

	err := i.Use(stdlib.Symbols)
	if err != nil {
		m.logger.Errorf("failed to load std symbols: %v", err)
	}

	err = i.Use(symbols.Symbols)
	if err != nil {
		m.logger.Errorf("failed to load self symbols: %v", err)
	}

	m.interpreter = i

	m.supportPlugins = map[string]types.Step{
		buildin.StepNameCMD:       &buildin.RunCmdStep{},
		buildin.StepNameShell:     &buildin.RunShellStep{},
		buildin.StepNameFile:      &buildin.FileUploadStep{},
		buildin.StepMultiNameFile: &buildin.MultiFileUploadStep{},
		buildin.StepNameZipFile:   &buildin.ZipFileStep{},
		buildin.StepNameYamlJson:  &buildin.JsonYamlReplaceStep{},
	}

	m.ReloadAllFilePlugins(m.pluginPath)
}

func (m *Manager) GetSSHList() *cache.Cache {
	return m.sshPoll
}

func (m *Manager) GetFileList() *utils.SafeMap {
	return m.fileList
}

func (m *Manager) NewClient(host *models.Host) (*transport.Client, error) {
	var (
		err       error
		cli       *transport.Client
		newStatus = host.Status
	)

	defer func() {
		if cli == nil && err != nil {
			newStatus = false
		}
		if host.Status != newStatus {
			host.Status = newStatus
			_ = models.UpdateHostStatus(host)
		}
	}()

	var c = &transport.ClientConfig{
		ID:       host.Id,
		Host:     host.Addr,
		Port:     host.Port,
		User:     host.User,
		Password: host.PassWord,
	}
	if host.PrivateKeyID != 0 {
		privateKey, err := models.GetPrivateKeyById(host.PrivateKeyID)
		if err != nil {
			m.logger.Errorf("error when get private key")
			return nil, err
		}
		c.KeyBytes = []byte(privateKey.KeyFile)
		c.Passphrase = privateKey.Passphrase
	}
	if cli, ok := m.sshPoll.Get(c.Serialize()); ok {
		err := cli.(*transport.Client).Ping()

		if err != nil {
			m.sshPoll.Remove(c.Serialize())
		} else {
			newStatus = true
			return cli.(*transport.Client), nil
		}
	}

	cli, err = transport.New(c)
	if err != nil {
		return nil, err
	}
	m.sshPoll.Add(c.Serialize(), cli)

	// notify when client close
	cli.Notify(m.statusChan)

	newStatus = true

	return cli, nil
}

func (m *Manager) GetStatus(host *models.Host) bool {
	client, err := m.NewClient(host)
	if err != nil {
		host.Status = false
		_ = models.UpdateHostStatus(host)
		return false
	}
	err = client.Ping()
	if err != nil {
		host.Status = false
		_ = models.UpdateHostStatus(host)
		return false
	}
	host.Status = true
	_ = models.UpdateHostStatus(host)
	return true
}

func (m *Manager) RemoveCache(host *models.Host) {
	m.removeCache(utils.InetAtoN(host.Addr, host.Port))
}

func (m *Manager) removeCache(inet int64) {
	m.sshPoll.Remove(inet)
}

func (m *Manager) GetAllPluginSchema() []Schema {
	var ret []Schema

	var keys []string
	for k := range m.supportPlugins {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		plugin := m.supportPlugins[k]
		sc, err := plugin.GetSchema()
		if err != nil {
			m.logger.Errorf("error when get plugin: %s scheme, err: %v", plugin.Name(), err)
			continue
		}
		ret = append(ret, Schema{
			Type:   plugin.Name(),
			Desc:   plugin.Desc(),
			Schema: sc,
		})
	}

	return ret
}

func (m *Manager) NewStep(typ string, id string, conf []byte) (types.Step, error) {
	for _, plugin := range m.supportPlugins {
		if plugin.Name() == typ {
			p, err := plugin.Create(conf)
			if err != nil {
				return nil, err
			}
			return p, nil
		}
	}

	return nil, errors.New("can not found step type")
}

func (m *Manager) ParseSteps(params string) ([]types.Step, error) {
	var (
		modSteps []models.Step
		steps    []types.Step
	)

	err := json.Unmarshal([]byte(params), &modSteps)
	if err != nil {
		return nil, err
	}

	sort.Sort(models.StepSlice(modSteps))

	for _, ms := range modSteps {
		step, setupErr := m.NewStep(ms.Type, ms.Name, []byte(ms.Params))
		if setupErr != nil {
			m.logger.Errorf("Error when new step with config, err: %v", setupErr)
			continue
		}
		steps = append(steps, step)
	}

	return steps, nil
}

func RunTaskWithQuit(client *transport.Client, cmd string, quitCh chan bool, writer io.Writer) (err error) {
	session, err := client.NewPty()
	if err != nil {
		return err
	}
	session.SetStdout(writer)

	errChan := make(chan error)
	go func(c string) {
		err = session.Run(c)
		if err != nil {
			errChan <- err
		}
		// can quit
		quitCh <- true
	}(cmd)
	defer session.Close()

	select {
	case <-quitCh:
		return
	case err := <-errChan:
		return err
	}
}

func (m *Manager) checkPlugin(manifest *Manifest) (types.Step, error) {

	_, err := m.interpreter.Eval(fmt.Sprintf(`import "%s"`, manifest.Import))
	if err != nil {
		return nil, err
	}

	createFn, err := m.interpreter.Eval(fmt.Sprintf(`%s.New`, manifest.Import))
	if err != nil {
		return nil, err
	}

	results := createFn.Call(nil)
	if len(results) != 1 {
		return nil, fmt.Errorf("invalid number of return for the New function: %d", len(results))
	}
	step, ok := results[0].Interface().(types.Step)
	if !ok {
		return nil, fmt.Errorf("invalid handler type: %T", results[0].Interface())
	}

	return step, nil
}
