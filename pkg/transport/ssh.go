package transport

import (
	"context"
	"errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"net"
	"oms/pkg/utils"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	DefaultTimeoutSec   = utils.GetEnvInt("ENV_SSH_DIAL_TIMEOUT", defaultTimeout)
	DefaultRWTimeoutSec = utils.GetEnvInt("ENV_SSH_RW_TIMEOUT", defaultRWTimeout)

	SSHDialTimeout = time.Duration(DefaultTimeoutSec) * time.Second
	SSHRWTimeout   = time.Duration(DefaultRWTimeoutSec) * time.Second
)

const (
	defaultTimeout   = 30
	defaultRWTimeout = 20
	KillSignal       = "0x09"
	KeepAliveMessage = "keepalive@golang.org"

	GOOSLinux   = "linux"
	GOOSFreeBSD = "freebsd"
	GOOSWindows = "windows"
	GOOSDarwin  = "darwin"
	GOOSUnknown = "unknown"

	ArchAmd64   = "amd64"
	ArchI386    = "386"
	ArchArm     = "arm"
	ArchUnknown = "unknown"

	DefaultPtyCols = 200
	DefaultPtyRows = 40
)

var gauge Gauge

type Gauge interface {
	Set(float64)
	Inc()
	Dec()
}

type MachineInfo struct {
	Goos string
	Arch string
	Cmd  string
}

type ClientConfig struct {
	ID         int    `json:"-"`
	Host       string `json:"host"`
	User       string `json:"user"`
	Password   string `json:"password"`
	Passphrase string `json:"passphrase"`
	KeyBytes   []byte `json:"key_bytes"`
	Port       int    `json:"port"`
}

func (h *ClientConfig) Serialize() int64 {
	return utils.InetAtoN(h.Host, h.Port)
}

type conn struct {
	net.Conn
}

func (c *conn) Write(b []byte) (n int, err error) {
	err = c.Conn.SetWriteDeadline(time.Now().Add(SSHRWTimeout))
	if err != nil {
		return 0, err
	}

	return c.Conn.Write(b)
}

func (c *conn) Read(b []byte) (n int, err error) {
	err = c.Conn.SetReadDeadline(time.Now().Add(SSHRWTimeout))
	if err != nil {
		return 0, err
	}

	return c.Conn.Read(b)
}

func newConn(c net.Conn) *conn {
	return &conn{
		c,
	}
}

type Client struct {
	conn       net.Conn
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	closer     chan struct{}
	Info       *MachineInfo
	Conf       *ClientConfig
	notify     []chan *Client
	mu         sync.Mutex
}

type Session struct {
	Client     *Client
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

func (c *Client) GetTargetMachineOs() string {
	return c.Info.Goos
}

func (c *Client) CollectTargetMachineInfo() error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	uName, err := session.Output("uname -a")
	if err != nil {
		session2, err := c.NewSession()
		if err != nil {
			return err
		}
		defer session2.Close()
		// todo windows arm & other os
		wmic, err := session2.Output("wmic os get OSArchitecture")
		if strings.Contains(string(wmic), "64") {
			c.Info.Arch = ArchAmd64
			c.Info.Goos = GOOSWindows
		} else if strings.Contains(string(wmic), "32") {
			c.Info.Arch = ArchI386
			c.Info.Goos = GOOSWindows
		}
		return nil
	}
	args := strings.Fields(string(uName))
	if len(args) < 2 {
		return errors.New("uname return an error length msg")
	}
	// todo freebsd & arm
	switch args[len(args)-1] {
	case "GNU/Linux", "Linux":
		c.Info.Goos = GOOSLinux
	case "FreeBSD":
		c.Info.Goos = GOOSFreeBSD
	}
	switch args[len(args)-2] {
	case "x86_64", "amd64":
		c.Info.Arch = ArchAmd64
	case "i386":
		c.Info.Arch = ArchI386
	case "armv6l", "armv7l":
		c.Info.Arch = ArchArm
	}

	return nil
}

func (c *Client) newSession() (*ssh.Session, error) {
	return c.sshClient.NewSession()
}

func (c *Client) NewSessionWithPty(cols, rows int) (*Session, error) {
	session, err := c.newSession()
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
		Client:     c,
		sshSession: session,
		stdin:      stdin,
	}, nil
}

func (c *Client) NewPty() (*Session, error) {
	return c.NewSessionWithPty(DefaultPtyCols, DefaultPtyRows)
}

func (c *Client) NewSession() (*Session, error) {
	session, err := c.newSession()
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
		Client:     c,
		sshSession: session,
		stdin:      stdin,
	}, nil
}

func (c *Client) GetSSHClient() *ssh.Client {
	return c.sshClient
}

func (c *Client) Ping() error {
	_, _, err := c.SendRequest(KeepAliveMessage, true, nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) keepAlive() {
	defer func() {
		c.mu.Lock()
		if c.notify != nil && len(c.notify) > 0 {
			for idx, _ := range c.notify {
				c.notify[idx] <- c
			}
		}
		c.mu.Unlock()
	}()

	const keepAliveInterval = 15 * time.Second
	t := time.NewTicker(keepAliveInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			err := c.Ping()
			if err != nil {
				return
			}
		case <-c.closer:
			return
		}
	}
}

func (c *Client) Notify(ch chan *Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for idx, _ := range c.notify {
		if c.notify[idx] == ch {
			return
		}
	}
	c.notify = append(c.notify, ch)
}

func (c *Client) SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error) {
	return c.sshClient.SendRequest(name, wantReply, payload)
}

func (c *Client) Close() error {
	c.closer <- struct{}{}

	return c.conn.Close()
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
	return s.sshSession.CombinedOutput(cmd)
}

// OutputContext run done and return output with context
func (s *Session) OutputContext(ctx context.Context, cmd string) ([]byte, error) {
	quit := make(chan struct{}, 1)
	defer close(quit)

	go func() {
		defer s.Close()
		select {
		case <-ctx.Done():
			return
		case <-quit:
			return
		}
	}()
	return s.sshSession.CombinedOutput(cmd)
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

func Dial(network, addr string, config *ssh.ClientConfig) (net.Conn, *ssh.Client, error) {
	conn, err := net.DialTimeout(network, addr, config.Timeout)
	if err != nil {
		return nil, nil, err
	}

	conn = newConn(conn)

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, nil, err
	}
	return conn, ssh.NewClient(c, chans, reqs), nil
}

// New 创建SSH client
func New(config *ClientConfig) (client *Client, err error) {
	clientConfig := &ssh.ClientConfig{
		User:            config.User,
		Timeout:         SSHDialTimeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 忽略public key的安全验证
	}

	if config.Port == 0 {
		config.Port = 22
	}

	// 1. private key bytes
	if config.KeyBytes != nil {
		if auth, err := AuthWithPrivateKeyBytes(config.KeyBytes, config.Passphrase); err == nil {
			clientConfig.Auth = append(clientConfig.Auth, auth)
		}
	}
	// 2. 密码方式 放在key之后,这样密钥失败之后可以使用Password方式
	if config.Password != "" {
		clientConfig.Auth = append(clientConfig.Auth, ssh.Password(config.Password))
	}
	// 3. agent 模式放在最后,这样当前两者都不能使用时可以采用Agent模式
	if auth, err := AuthWithAgent(); err == nil {
		clientConfig.Auth = append(clientConfig.Auth, auth)
	}

	conn, sshClient, err := Dial("tcp", net.JoinHostPort(config.Host, strconv.Itoa(config.Port)), clientConfig)

	if err != nil {
		return client, errors.New("Failed to dial ssh: " + err.Error())
	}
	client = &Client{
		conn:      conn,
		sshClient: sshClient,
		closer:    make(chan struct{}, 1),
		Conf:      config,
		mu:        sync.Mutex{},
		Info: &MachineInfo{
			Goos: GOOSUnknown,
			Arch: ArchUnknown,
		},
	}

	err = client.CollectTargetMachineInfo()
	if err != nil {
		return nil, err
	}

	go client.keepAlive()
	return client, nil
}
