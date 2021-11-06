package transport

import (
	"bytes"
	"errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

/*
变量声明
*/

const (
	DefaultTimeout = 30 * time.Second
	KillSignal     = "0x09"
)

var gauge Gauge

type Gauge interface {
	Set(float64)
	Inc()
	Dec()
}

type Client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

type Session struct {
	sshSession *ssh.Session
	stdin      io.WriteCloser
}

/*
ssh基础服务
*/

// RegisterSessionGauge call register a gauge listen session num
func RegisterSessionGauge(g Gauge) {
	gauge = g
}

func (c *Client) NewSessionWithPty(cols, rows int) (*Session, error) {
	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	if gauge != nil {
		gauge.Inc()
	}
	stdin, err := session.StdinPipe()
	if err != nil {
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
	if gauge != nil {
		gauge.Inc()
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return nil, err
	}
	return &Session{
		sshSession: session,
		stdin:      stdin,
	}, nil
}

func (c *Client) GetSSHClient() *ssh.Client {
	return c.sshClient
}

func (s *Session) Kill() error {
	// kill signal
	if _, err := s.Write([]byte(KillSignal)); err != nil {
		return err
	}
	return nil
}

func (s *Session) Close() error {
	if gauge != nil {
		gauge.Dec()
	}
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
	var stderr bytes.Buffer
	s.SetStderr(&stderr)

	msg, err := s.sshSession.Output(cmd)
	if err != nil {
		return stderr.Bytes(), err
	} else {
		return msg, nil
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

// New 创建SSH client
func New(host, user, password string, KeyBytes []byte, port int) (client *Client, err error) {
	clientConfig := &ssh.ClientConfig{
		User:            user,
		Timeout:         DefaultTimeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略public key的安全验证
	}

	if port == 0 {
		port = 22
	}

	// 1. private key bytes
	if KeyBytes != nil {
		if auth, err := AuthWithPrivateKeyBytes(KeyBytes, password); err == nil {
			clientConfig.Auth = append(clientConfig.Auth, auth)
		}
	}
	// 2. 密码方式 放在key之后,这样密钥失败之后可以使用Password方式
	if password != "" {
		clientConfig.Auth = append(clientConfig.Auth, ssh.Password(password))
	}
	// 3. agent 模式放在最后,这样当前两者都不能使用时可以采用Agent模式
	if auth, err := AuthWithAgent(); err == nil {
		clientConfig.Auth = append(clientConfig.Auth, auth)
	}

	sshClient, err := ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), clientConfig)

	if err != nil {
		return client, errors.New("Failed to dial ssh: " + err.Error())
	}

	return &Client{sshClient: sshClient}, nil
}
