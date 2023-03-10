package websocket

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/ssbeatty/oms/internal/ssh"
	"github.com/ssbeatty/oms/pkg/logger"
	"github.com/ssbeatty/oms/pkg/transport"
	"sync"
	"time"
)

const (
	minSizeOfResizeMsg = 12
)

type wsBufferWriter struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

// implement Write interface to write bytes from ssh server into bytes.Buffer.
func (w *wsBufferWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

func flushComboOutput(w *wsBufferWriter, wsConn *websocket.Conn) error {
	if w.buffer.Len() != 0 {
		err := wsConn.WriteMessage(websocket.BinaryMessage, w.buffer.Bytes())
		if err != nil {
			return err
		}
		w.buffer.Reset()
	}
	return nil
}

// SSHSession connect to ssh server using ssh session.
type SSHSession struct {
	*transport.Session
	once        sync.Once
	logger      *logger.Logger
	comboOutput *wsBufferWriter
}

func NewSshConn(cols, rows int, sshClient *transport.Client) (*SSHSession, error) {
	sshSession, err := sshClient.NewSessionWithPty(cols, rows)
	if err != nil {
		return nil, err
	}
	if err := sshSession.Shell(); err != nil {
		return nil, err
	}
	comboWriter := new(wsBufferWriter)
	sshSession.SetStderr(comboWriter)
	sshSession.SetStdout(comboWriter)

	return &SSHSession{
		once:        sync.Once{},
		Session:     sshSession,
		comboOutput: comboWriter,
		logger:      logger.NewLogger("webSSH"),
	}, nil
}

func (s *SSHSession) Close() {
	if s.Session != nil {
		s.Session.Close()
	}

}

//ReceiveWsMsg  receive websocket msg do some handling then write into ssh.session.stdin
func (s *SSHSession) ReceiveWsMsg(wsConn *websocket.Conn, exitCh chan struct{}) {
	//tells other go routine quit
	defer s.setQuit(exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			// read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				s.logger.Errorf("reading webSocket message failed, err: %v", err)
				return
			}

			// 每次传输一个或多个char
			if len(wsData) > minSizeOfResizeMsg {
				// resize 或者 粘贴
				resize := ssh.WindowSize{}
				err := json.Unmarshal(wsData, &resize)
				if err != nil {
					s.logger.Errorf("unmarshal resize error: %v", err)
					// 粘贴内容
					goto SEND
				}
				if resize.Cols > 0 && resize.Rows > 0 {
					if err := s.Session.WindowChange(resize.Rows, resize.Cols); err != nil {
						s.logger.Errorf("ssh pty change windows size failed, err: %v", err)
					}
				}
				break
			}
		SEND:
			decodeBytes := wsData
			if _, err := s.Session.Write(decodeBytes); err != nil {
				s.logger.Errorf("ws cmd bytes write to ssh.stdin pipe failed, err: %v", err)
			}
		}
	}
}

func (s *SSHSession) SendComboOutput(wsConn *websocket.Conn, exitCh chan struct{}) {
	//tells other go routine quit
	defer s.setQuit(exitCh)

	//every 120ms write combine output bytes into websocket response
	tick := time.NewTicker(time.Millisecond * time.Duration(120))

	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			//write combine output bytes into websocket response
			if err := flushComboOutput(s.comboOutput, wsConn); err != nil {
				return
			}
		case <-exitCh:
			return
		}
	}
}

func (s *SSHSession) SessionWait(quitChan chan struct{}) {
	if err := s.Session.Wait(); err != nil {
		s.logger.Errorf("ssh session wait failed, err: %v", err)
		s.setQuit(quitChan)
	}
}

func (s *SSHSession) setQuit(ch chan struct{}) {
	s.once.Do(func() {
		close(ch)
	})
}
