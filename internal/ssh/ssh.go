package ssh

import (
	"encoding/json"
	"github.com/ssbeatty/oms/internal/config"
	"github.com/ssbeatty/oms/internal/models"
	"github.com/ssbeatty/oms/internal/ssh/buildin"
	"github.com/ssbeatty/oms/pkg/cache"
	"github.com/ssbeatty/oms/pkg/logger"
	"github.com/ssbeatty/oms/pkg/transport"
	"github.com/ssbeatty/oms/pkg/utils"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
)

const (
	CMDTypeShell  = "cmd"
	CMDTypePlayer = "player"

	PollSize = 2000
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
	supportPlugins map[string]buildin.Step // 注册所有的插件类型通过接口返回 表单渲染的格式
	cfg            *config.Conf
	statusChan     chan *transport.Client
}

func NewManager(cfg *config.Conf) *Manager {
	return &Manager{
		notify:     make(chan bool, 10),
		subClients: sync.Map{},
		fileList:   utils.NewSafeMap(),
		sshPoll:    cache.NewCache(PollSize),
		logger:     logger.NewLogger("sshManager"),
		cfg:        cfg,
		statusChan: make(chan *transport.Client),
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
		if info.IsDir() {
			return nil
		}
		name, err := checkPlugin(path)
		if err != nil {
			return nil
		}

		m.supportPlugins[name] = &buildin.PluginStep{
			ScriptPath: path,
		}
		return nil
	})
}

// initAllPlugins 启动时加载所有插件
func (m *Manager) initAllPlugins() {
	pluginPath := filepath.Join(m.cfg.App.DataPath, config.PluginPath)
	if exists, _ := utils.PathExists(pluginPath); !exists {
		_ = os.MkdirAll(pluginPath, fs.ModePerm)
	}

	m.supportPlugins = map[string]buildin.Step{
		buildin.StepNameCMD:       &buildin.RunCmdStep{},
		buildin.StepNameShell:     &buildin.RunShellStep{},
		buildin.StepNameFile:      &buildin.FileUploadStep{},
		buildin.StepMultiNameFile: &buildin.MultiFileUploadStep{},
		buildin.StepNameZipFile:   &buildin.ZipFileStep{},
		buildin.StepNameYamlJson:  &buildin.JsonYamlReplaceStep{},
	}

	m.ReloadAllFilePlugins(pluginPath)
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
		sc, err := plugin.GetSchema(plugin)
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

func (m *Manager) NewStep(typ string) buildin.Step {
	for _, plugin := range m.supportPlugins {
		if plugin.Name() == typ {
			return plugin.Create()
		}
	}

	return nil
}

func (m *Manager) ParseSteps(params string) ([]buildin.Step, error) {
	var (
		modSteps []models.Step
		steps    []buildin.Step
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
	name, err := exec.Command(path, buildin.CMDName).CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(name), nil
}
