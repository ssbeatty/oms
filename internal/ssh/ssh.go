package ssh

import (
	"encoding/json"
	"io"
	"io/fs"
	"oms/internal/config"
	"oms/internal/models"
	"oms/internal/utils"
	"oms/pkg/cache"
	"oms/pkg/logger"
	"oms/pkg/transport"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
)

const (
	PluginPath    = "plugin"
	CMDTypeShell  = "cmd"
	CMDTypePlayer = "player"
)

type Config struct {
	User       string
	Host       string
	Port       int
	Password   string
	KeyBytes   []byte
	Passphrase string
}

type Schema struct {
	Type   string      `json:"type"`
	Schema interface{} `json:"schema"`
}

type Command struct {
	Type   string
	Params string
}

type Result struct {
	Seq      int    `json:"seq"`
	Status   bool   `json:"status"`
	HostId   int    `json:"host_id"`
	HostName string `json:"hostname"`
	Msg      string `json:"msg"`
}

// 对host和port进行hash
func (h *Config) serialize() int64 {
	return utils.InetAtoN(h.Host, h.Port)
}

type Manager struct {
	fileList       *utils.SafeMap
	sshPoll        *cache.Cache
	logger         *logger.Logger
	notify         chan bool
	subClients     sync.Map
	supportPlugins map[string]Step // 注册所有的插件类型通过接口返回 表单渲染的格式
	cfg            *config.Conf
}

func NewManager(cfg *config.Conf) *Manager {
	return &Manager{
		notify:     make(chan bool, 10),
		subClients: sync.Map{},
		fileList:   utils.NewSafeMap(),
		sshPoll:    cache.NewCache(1000),
		logger:     logger.NewLogger("sshManager"),
		cfg:        cfg,
	}
}

func (m *Manager) Init() *Manager {
	go m.doNotifyFileTaskList()
	m.initAllPlugins()

	return m
}

// initAllPlugins 启动时加载所有插件
func (m *Manager) initAllPlugins() {
	pluginPath := filepath.Join(m.cfg.App.DataPath, PluginPath)
	if exists, _ := utils.PathExists(pluginPath); !exists {
		_ = os.MkdirAll(pluginPath, fs.ModePerm)
	}

	m.supportPlugins = map[string]Step{
		StepNameCMD:       &RunCmdStep{},
		StepNameShell:     &RunShellStep{},
		StepNameFile:      &FileUploadStep{},
		StepMultiNameFile: &MultiFileUploadStep{},
	}

	files, err := os.ReadDir(pluginPath)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		plg := filepath.Join(pluginPath, f.Name())
		name, err := checkPlugin(plg)
		if err != nil {
			m.logger.Errorf("error when check plugin: %s, err: %v", plg, err)
			continue
		}

		m.supportPlugins[name] = &PluginStep{
			ScriptPath: plg,
		}
	}

}

func (m *Manager) GetSSHList() *cache.Cache {
	return m.sshPoll
}

func (m *Manager) GetFileList() *utils.SafeMap {
	return m.fileList
}

func (m *Manager) NewClient(host *models.Host) (*transport.Client, error) {
	var c = &Config{
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
	if cli, ok := m.sshPoll.Get(c.serialize()); ok {
		ss, err := cli.(*transport.Client).NewSession()

		if err != nil {
			m.sshPoll.Remove(c.serialize())
		} else {
			defer ss.Close()
			return cli.(*transport.Client), nil
		}
	}

	cli, err := transport.New(c.Host, c.User, c.Password, c.Passphrase, c.KeyBytes, c.Port)
	if err != nil {
		return nil, err
	}
	m.sshPoll.Add(c.serialize(), cli)
	return cli, nil
}

func (m *Manager) GetStatus(host *models.Host) bool {
	client, err := m.NewClient(host)
	if err != nil {
		host.Status = false
		_ = models.UpdateHostStatus(host)
		return false
	}
	session, err := client.NewSession()
	if err != nil {
		host.Status = false
		_ = models.UpdateHostStatus(host)
		return false
	}
	defer session.Close()
	host.Status = true
	_ = models.UpdateHostStatus(host)
	return true
}

func (m *Manager) RemoveCache(host *models.Host) {
	m.sshPoll.Remove(utils.InetAtoN(host.Addr, host.Port))
}

func (m *Manager) GetAllPluginSchema() []Schema {
	var ret []Schema

	for _, plugin := range m.supportPlugins {
		sc, err := plugin.GetSchema(plugin)
		if err != nil {
			m.logger.Errorf("error when get plugin: %s scheme, err: %v", plugin.Name(), err)
			continue
		}
		ret = append(ret, Schema{
			Type:   plugin.Name(),
			Schema: sc,
		})
	}

	return ret
}

func (m *Manager) NewStep(typ string) Step {
	for _, plugin := range m.supportPlugins {
		if plugin.Name() == typ {
			return plugin.Create()
		}
	}

	return nil
}

func (m *Manager) ParseSteps(params string) ([]Step, error) {
	var (
		modSteps []models.Step
		steps    []Step
	)

	err := json.Unmarshal([]byte(params), &modSteps)
	if err != nil {
		return nil, err
	}

	sort.Sort(models.StepSlice(modSteps))

	for _, ms := range modSteps {
		step := m.NewStep(ms.Type)
		err := json.Unmarshal([]byte(ms.Params), step)
		if err != nil {
			return nil, err
		}
		step.SetID(ms.Name)
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

func checkPlugin(path string) (string, error) {
	name, err := exec.Command(path, CMDName).Output()
	if err != nil {
		return "", err
	}

	return string(name), nil
}
