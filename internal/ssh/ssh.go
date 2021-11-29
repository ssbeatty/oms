package ssh

import (
	"io"
	"oms/internal/models"
	"oms/internal/utils"
	"oms/pkg/cache"
	"oms/pkg/logger"
	"oms/pkg/transport"
	"sync"
)

type Config struct {
	User       string
	Host       string
	Port       int
	Password   string
	KeyBytes   []byte
	Passphrase string
}

// 对host和port进行hash
func (h *Config) serialize() int64 {
	return utils.InetAtoN(h.Host, h.Port)
}

type Manager struct {
	fileList   *utils.SafeMap
	sshPoll    *cache.Cache
	logger     *logger.Logger
	notify     chan bool
	subClients sync.Map
}

func NewManager() *Manager {
	return &Manager{
		notify:     make(chan bool, 10),
		subClients: sync.Map{},
		fileList:   utils.NewSafeMap(),
		sshPoll:    cache.NewCache(1000),
		logger:     logger.NewLogger("sshManager"),
	}
}

func (m *Manager) Init() *Manager {
	go m.doNotifyFileTaskList()

	return m
}

func (m *Manager) GetSSHList() *cache.Cache {
	return m.sshPoll
}

func (m *Manager) GetFileList() *utils.SafeMap {
	return m.fileList
}

func (m *Manager) NewClient(host *models.Host) (*transport.Client, error) {
	var config = &Config{
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
		config.KeyBytes = []byte(privateKey.KeyFile)
		config.Passphrase = privateKey.Passphrase
	}
	if cli, ok := m.sshPoll.Get(config.serialize()); ok {
		ss, err := cli.(*transport.Client).NewSession()

		if err != nil {
			m.sshPoll.Remove(config.serialize())
		} else {
			defer ss.Close()
			return cli.(*transport.Client), nil
		}
	}

	cli, err := transport.New(config.Host, config.User, config.Password, config.Passphrase, config.KeyBytes, config.Port)
	if err != nil {
		return nil, err
	}
	m.sshPoll.Add(config.serialize(), cli)
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
