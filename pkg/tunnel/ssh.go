/*
ssh 实现隧道 对应ssh命令的L和R参数
*/

package tunnel

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"sync"
	"sync/atomic"
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
	closer      chan bool
	isOpen      bool
	client      *ssh.Client
	listener    net.Listener
}

func NewSSHTunnel(client *ssh.Client, destination, source, mode string) *SSHTunnel {
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
		client:      client,
		isOpen:      true,
		closer:      make(chan bool),
	}
	sshTunnel.errorMsg.Store("success")

	return sshTunnel
}

func (s *SSHTunnel) newConnectionWaiter(c chan net.Conn) {
	conn, err := s.listener.Accept()
	if err != nil {
		log.Errorf("error when tunnel accept, err: %v", err)
		s.SetErrorMsg("listening error", err)
		return
	}
	c <- conn
}

func (s *SSHTunnel) Close() {
	close(s.closer)
}

func (s *SSHTunnel) Status() bool {
	return s.isOpen
}

func (s *SSHTunnel) GetErrorMsg() string {
	return s.errorMsg.Load().(string)
}

func (s *SSHTunnel) SetErrorMsg(msg string, err error) {
	s.errorMsg.Store(fmt.Sprintf("%s, err: %v", msg, err))
}

func (s *SSHTunnel) Start() {
	var err error
	var localListener net.Listener
	var once sync.Once

	if s.Mode == LocalMode {
		localListener, err = net.Listen("tcp", s.Destination)
	} else {
		localListener, err = s.client.Listen("tcp", s.Destination)
	}
	if err != nil {
		log.Errorf("error when tunnel listen: %s, err: %v", s.Destination, err)
		s.SetErrorMsg("listening error", err)
		return
	}

	s.listener = localListener

	for {
		if !s.isOpen {
			break
		}
		c := make(chan net.Conn)
		go s.newConnectionWaiter(c)

		once.Do(func() {
			log.Infof("tunnel src: '%s', dest: '%s' connect success.", s.Source, s.Destination)
		})

		select {
		case <-s.closer:
			s.isOpen = false
		case localConn := <-c:
			go s.forward(localConn)
		}
	}
	_ = s.listener.Close()
	log.Infof("tunnel src: '%s', dest: '%s' closed.", s.Source, s.Destination)
}

func (s *SSHTunnel) forward(localConn net.Conn) {
	var err error
	var remoteConn net.Conn
	if s.Mode == LocalMode {
		remoteConn, err = s.client.Dial("tcp", s.Source)
	} else {
		remoteConn, err = net.Dial("tcp", s.Source)
	}
	if err != nil {
		log.Errorf("error when dial local source, err: %v", err)
		s.SetErrorMsg("dial error", err)
		return
	}

	defer remoteConn.Close()
	defer localConn.Close()

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if errors.Is(err, io.EOF) {
			return
		}
	}
	// close when EOF
	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)

	log.Debug("new conn forward success.")
	<-s.closer
}
