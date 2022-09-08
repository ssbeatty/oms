/*
ssh 实现隧道 对应ssh命令的L和R参数
*/

package tunnel

import (
	"fmt"
	"io"
	"net"
	"oms/pkg/logger"
	"oms/pkg/transport"
	"sync"
	"sync/atomic"
	"time"
)

const (
	LocalMode  = "local"
	RemoteMode = "remote"
)

type SSHTunnel struct {
	Mode        string
	Source      string
	Destination string
	errorMsg    atomic.Value
	closer      chan struct{}
	isOpen      atomic.Value
	listener    net.Listener
	logger      *logger.Logger
	conf        *transport.ClientConfig
	client      *transport.Client
}

func NewSSHTunnel(conf *transport.ClientConfig, destination, source, mode string) *SSHTunnel {
	var realMode string
	switch mode {
	case RemoteMode:
		realMode = RemoteMode
	default:
		realMode = LocalMode
	}
	sshTunnel := &SSHTunnel{
		Mode:        realMode,
		Source:      source,
		Destination: destination,
		conf:        conf,
		closer:      make(chan struct{}),
		logger:      logger.NewLogger("tunnel"),
	}

	sshTunnel.errorMsg.Store("")
	sshTunnel.isOpen.Store(false)

	return sshTunnel
}

func (s *SSHTunnel) setOpened(status bool) {
	s.isOpen.Store(status)
}

func (s *SSHTunnel) Close() {
	s.setOpened(false)

	close(s.closer)
}

func (s *SSHTunnel) Status() bool {
	return s.isOpen.Load().(bool)
}

func (s *SSHTunnel) GetErrorMsg() string {
	return s.errorMsg.Load().(string)
}

func (s *SSHTunnel) SetErrorMsg(msg string, err error) {
	s.errorMsg.Store(fmt.Sprintf("%s, err: %v", msg, err))
}

// 重置start方法
func (s *SSHTunnel) reset() {
	s.setOpened(false)
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func (s *SSHTunnel) manage() {
	beatTicker := time.NewTicker(10 * time.Second)
	manageTicker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-manageTicker.C:
			if !s.Status() && s.listener == nil {
				go s.start()
			}
		case <-beatTicker.C:
			if s.client != nil {
				err := s.client.Ping()
				if err != nil {
					s.reset()
				}
			}
		case <-s.closer:
			return
		}
	}
}

func (s *SSHTunnel) start() {
	var (
		err           error
		once          sync.Once
		localListener net.Listener
	)

	s.setOpened(true)

	defer func() {
		s.setOpened(false)

		if s.listener != nil {
			_ = s.listener.Close()
			s.listener = nil
		}

		if s.client != nil {
			_ = s.client.Close()
			s.client = nil
		}
	}()

	if s.client == nil {
		s.client, err = transport.New(s.conf)
		if err != nil {
			s.logger.Errorf("error when tunnel create ssh client: %v", err)
			return
		}
	}

	if s.Mode == LocalMode {
		localListener, err = net.Listen("tcp", s.Destination)
	} else {
		localListener, err = s.client.GetSSHClient().Listen("tcp", s.Destination)
	}
	if err != nil {
		s.logger.Errorf("error when tunnel listen: %s, err: %v", s.Destination, err)
		s.SetErrorMsg("listening error", err)
		return
	}

	s.errorMsg.Store("success")

	s.listener = localListener

	for {
		if !s.Status() {
			break
		}

		once.Do(func() {
			s.logger.Infof("tunnel src: '%s', dest: '%s' connect success.", s.Source, s.Destination)
		})

		conn, err := s.listener.Accept()
		if err != nil {
			s.logger.Errorf("error when tunnel accept, err: %v", err)
			s.SetErrorMsg("listening error", err)
			return
		}
		go s.forward(conn)
	}

	s.logger.Infof("tunnel src: '%s', dest: '%s' closed.", s.Source, s.Destination)
}

func (s *SSHTunnel) Start() {
	go s.start()

	s.manage()
}

func (s *SSHTunnel) forward(localConn net.Conn) {
	var err error
	var remoteConn net.Conn
	if s.Mode == LocalMode {
		remoteConn, err = s.client.GetSSHClient().Dial("tcp", s.Source)
	} else {
		remoteConn, err = net.Dial("tcp", s.Source)
	}
	if err != nil {
		s.logger.Errorf("error when dial local source, err: %v", err)
		s.SetErrorMsg("dial error", err)
		s.reset()
		return
	}

	copyConn := func(writer, reader net.Conn) {
		defer writer.Close()
		defer reader.Close()

		_, err := io.Copy(writer, reader)
		if err != nil {
			return
		}
	}
	// 关闭连接后退出
	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)

	s.logger.Debug("new conn forward success.")
}
