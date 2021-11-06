package ssh

import (
	"io"
	"oms/internal/models"
	"oms/internal/utils"
	"oms/pkg/cache"
	"oms/pkg/logger"
	"oms/pkg/transport"
)

type Config struct {
	User       string
	Host       string
	Port       int
	Password   string
	KeyBytes   []byte
	Passphrase string
}

func (h *Config) serialize() int64 {
	return utils.InetAtoN(h.Host, h.Port)
}

type Manager struct {
	fileList *utils.SafeMap
	sshPoll  *cache.Cache
	logger   *logger.Logger
}

func NewManager() *Manager {
	return &Manager{
		fileList: utils.NewSafeMap(),
		sshPoll:  cache.NewCache(1000),
		logger:   logger.NewLogger("sshManager"),
	}
}

func (m *Manager) GetSSHList() *cache.Cache {
	return m.sshPoll
}

func (m *Manager) GetFileList() *utils.SafeMap {
	return m.fileList
}

func (m *Manager) NewClient(host string, port int, user string, password string, KeyBytes []byte) (*transport.Client, error) {
	if user == "" {
		user = "root"
	}
	var config = &Config{
		Host:       host,
		Port:       port,
		User:       user,
		Password:   password,
		Passphrase: password,
	}
	if KeyBytes != nil {
		config.KeyBytes = KeyBytes
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

	cli, err := transport.New(config.Host, config.User, config.Password, config.KeyBytes, config.Port)
	if err != nil {
		return nil, err
	}
	m.sshPoll.Add(config.serialize(), cli)
	return cli, nil
}

func (m *Manager) GetStatus(host *models.Host) bool {
	client, err := m.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
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
	session, err := client.NewSessionWithPty(20, 20)
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
	}(cmd)
	defer session.Close()

	select {
	case <-quitCh:
		return
	case err := <-errChan:
		return err
	}
}
