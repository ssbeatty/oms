package transport

import (
	"errors"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

const (
	DefaultTimeout = 30 * time.Second
	KillSignal     = "0x09"
)

type Config struct {
	User       string
	Host       string
	Port       int
	Password   string
	KeyBytes   []byte
	Passphrase string
}

type Client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

type Session struct {
	sshSession *ssh.Session
	stdin      io.WriteCloser
}

func (c *Client) NewSessionWithPty(cols, rows int) (*Session, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Debugf("get stdin pipe error, %v", err)
		return nil, err
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // disable echo
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := session.RequestPty("xterm", rows, cols, modes); err != nil {
		return nil, err
	}
	return &Session{
		sshSession: session,
		stdin:      stdin,
	}, nil
}

func (c *Client) NewSession() (*Session, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		log.Debugf("get stdin pipe error, %v", err)
		return nil, err
	}
	return &Session{
		sshSession: session,
		stdin:      stdin,
	}, nil
}

func (s *Session) Kill() error {
	// kill signal
	//if _, err := s.Write([]byte(KillSignal)); err != nil {
	//	return err
	//}
	return nil
}

func (s *Session) Close() error {
	return s.sshSession.Close()
}

func (s *Session) WindowChange(h, w int) error {
	return s.sshSession.WindowChange(h, w)
}

func (s *Session) Start(cmd string) error {
	return s.sshSession.Start(cmd)
}

func (s *Session) Wait() error {
	return s.sshSession.Wait()
}

func (s *Session) Run(cmd string) error {
	return s.sshSession.Run(cmd)
}

func (s *Session) Shell() error {
	return s.sshSession.Shell()
}

func (s *Session) SetStderr(stderr io.Writer) {
	s.sshSession.Stderr = stderr
}

func (s *Session) SetStdout(stdout io.Writer) {
	s.sshSession.Stdout = stdout
}

func (s *Session) Write(b []byte) (int, error) {
	return s.stdin.Write(b)
}

// Output run done and return output
func (s *Session) Output(cmd string) ([]byte, error) {
	return s.sshSession.Output(cmd)
}

func (s *Session) RunTaskWithQuit(cmd string, quitCh <-chan bool) {
	go func(c string) {
		err := s.sshSession.Run(c)
		if err != nil {
			log.Errorf("RunTaskWithQuit run task error, %v", err)
		}
	}(cmd)

	select {
	case <-quitCh:
		// 理论上pty下不需要kill
		if err := s.Kill(); err != nil {
			return
		}
		if err := s.Close(); err != nil {
			return
		}
	}
}

// AuthWithAgent use already authed user
func AuthWithAgent() (ssh.AuthMethod, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, errors.New("agent disabled")
	}
	socks, err := net.Dial("unix", sock)
	if err != nil {
		return nil, err
	}
	// 1. 返回Signers函数的结果
	client := agent.NewClient(socks)
	signers, err := client.Signers()
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signers...), nil
}

// AuthWithPrivateKeyBytes 直接通过秘钥的bytes
func AuthWithPrivateKeyBytes(key []byte, password string) (ssh.AuthMethod, error) {
	var signer ssh.Signer
	var err error
	if password == "" {
		signer, err = ssh.ParsePrivateKey(key)
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(password))
	}
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

func NewClientWithSftp(host string, port int, user string, password string, KeyBytes []byte) (*Client, error) {
	client, err := NewClient(host, port, user, password, KeyBytes)
	if err != nil {
		return nil, err
	}
	err = client.NewSftpClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}

// New 创建SSH client
func New(cnf *Config) (client *Client, err error) {
	clientConfig := &ssh.ClientConfig{
		User:            cnf.User,
		Timeout:         DefaultTimeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略public key的安全验证
	}

	if cnf.Port == 0 {
		cnf.Port = 22
	}

	// 1. private key bytes
	if cnf.KeyBytes != nil {
		if auth, err := AuthWithPrivateKeyBytes(cnf.KeyBytes, cnf.Password); err == nil {
			clientConfig.Auth = append(clientConfig.Auth, auth)
		}
	}
	// 2. 密码方式 放在key之后,这样密钥失败之后可以使用Password方式
	if cnf.Password != "" {
		clientConfig.Auth = append(clientConfig.Auth, ssh.Password(cnf.Password))
	}
	// 3. agent 模式放在最后,这样当前两者都不能使用时可以采用Agent模式
	if auth, err := AuthWithAgent(); err == nil {
		clientConfig.Auth = append(clientConfig.Auth, auth)
	}

	sshClient, err := ssh.Dial("tcp", net.JoinHostPort(cnf.Host, strconv.Itoa(cnf.Port)), clientConfig)

	if err != nil {
		return client, errors.New("Failed to dial ssh: " + err.Error())
	}

	return &Client{sshClient: sshClient}, nil
}

func (c *Client) NewSftpClient() error {
	cli, err := sftp.NewClient(c.sshClient)
	if err != nil {
		return err
	}
	c.sftpClient = cli
	return nil
}

func (c *Client) ReadDir(path string) ([]os.FileInfo, error) {
	if c.IsDir(path) {
		info, err := c.sftpClient.ReadDir(path)
		return info, err
	}
	return nil, nil
}

func (c *Client) GetFile(path string) (*sftp.File, error) {
	file, err := c.sftpClient.Open(path)
	if err != nil {
		return nil, err
	}
	return file, err
}

func (c *Client) IsDir(path string) bool {
	// 检查远程是文件还是目录
	info, err := c.sftpClient.Stat(path)
	if err == nil && info.IsDir() {
		return true
	}
	return false
}

func (c *Client) MkdirAll(dirPath string) error {

	parentDir := filepath.ToSlash(filepath.Dir(dirPath))
	_, err := c.sftpClient.Stat(parentDir)
	if err != nil {
		// log.Println(err)
		if err.Error() == "file does not exist" {
			err := c.MkdirAll(parentDir)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	err = c.sftpClient.Mkdir(filepath.ToSlash(dirPath))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Remove(path string) error {
	return c.sftpClient.Remove(path)
}

func (c *Client) RemoveDir(remoteDir string) error {
	remoteFiles, err := c.sftpClient.ReadDir(remoteDir)
	if err != nil {
		return err
	}
	for _, file := range remoteFiles {
		subRemovePath := path.Join(remoteDir, file.Name())
		if file.IsDir() {
			c.RemoveDir(subRemovePath)
		} else {
			c.Remove(subRemovePath)
		}
	}
	c.sftpClient.RemoveDirectory(remoteDir)
	return nil
}

func (c *Client) ReadLink(path string) (string, error) {
	return c.sftpClient.ReadLink(path)
}

func (c *Client) Stat(path string) (os.FileInfo, error) {
	return c.sftpClient.Stat(path)
}

func (c *Client) RealPath(path string) (string, error) {
	return c.sftpClient.RealPath(path)
}

func (c *Client) GetPwd() (string, error) {
	return c.sftpClient.Getwd()
}
