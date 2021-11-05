package web

import (
	"bytes"
	"github.com/gorilla/websocket"
	"oms/pkg/logger"
	"oms/pkg/transport"
	"sync"
	"time"
)

const (
	wsMsgCmd    = "cmd"
	wsMsgResize = "resize"
)

type wsMsg struct {
	Type string `json:"type"`
	Cmd  string `json:"cmd"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

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
		err := wsConn.WriteMessage(websocket.TextMessage, w.buffer.Bytes())
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
func (s *SSHSession) ReceiveWsMsg(wsConn *websocket.Conn, exitCh chan bool) {
	//tells other go routine quit
	defer setQuit(exitCh)
	for {
		select {
		case <-exitCh:
			return
		default:
			//read websocket msg
			_, wsData, err := wsConn.ReadMessage()
			if err != nil {
				s.logger.Errorf("reading webSocket message failed, err: %v", err)
				return
			}
			// 按照字符发送的，所以用json代价太大了。。
			// unmarshal bytes into struct
			msgObj := wsMsg{
				Type: "cmd",
				Cmd:  "",
				Rows: 40,
				Cols: 180,
			}

			switch msgObj.Type {
			case wsMsgResize:
				if msgObj.Cols > 0 && msgObj.Rows > 0 {
					if err := s.Session.WindowChange(msgObj.Rows, msgObj.Cols); err != nil {
						s.logger.Errorf("ssh pty change windows size failed, err: %v", err)
					}
				}
			case wsMsgCmd:
				decodeBytes := wsData
				if err != nil {
					s.logger.Errorf("websocket cmd string base64 decoding failed, err: %v", err)
				}
				if _, err := s.Session.Write(decodeBytes); err != nil {
					s.logger.Errorf("ws cmd bytes write to ssh.stdin pipe failed, err: %v", err)
				}
			}
		}
	}
}

func (s *SSHSession) SendComboOutput(wsConn *websocket.Conn, exitCh chan bool) {
	//tells other go routine quit
	defer setQuit(exitCh)

	//every 120ms write combine output bytes into websocket response
	tick := time.NewTicker(time.Millisecond * time.Duration(120))
	//for range time.Tick(120 * time.Millisecond){}
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

func (s *SSHSession) SessionWait(quitChan chan bool) {
	if err := s.Session.Wait(); err != nil {
		s.logger.Errorf("ssh session wait failed, err: %v", err)
		setQuit(quitChan)
	}
}

func setQuit(ch chan bool) {
	ch <- true
}
