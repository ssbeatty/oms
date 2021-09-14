package wscontrol

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"io"
	"oms/transport"
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

// SSHSession connect to ssh server using ssh session.
type SSHSession struct {
	*transport.Session
}

type WSConnect struct {
	*websocket.Conn
}

func (w *WSConnect) Write(p []byte) (int, error) {
	log.Println(string(p))
	err := w.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func NewSshConn(cols, rows int, sshClient *transport.Client) (*SSHSession, error) {
	sshSession, err := sshClient.NewSessionWithPty(cols, rows)
	if err != nil {
		return nil, err
	}
	if err := sshSession.SSHSession.Shell(); err != nil {
		return nil, err
	}
	return &SSHSession{Session: sshSession}, nil
}

func (s *SSHSession) Close() {
	if s.Session != nil {
		s.Session.Close()
	}

}

func (s *SSHSession) SetOutput(closer io.WriteCloser) {
	s.Session.SSHSession.Stderr = closer
	s.Session.SSHSession.Stdout = closer
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
				log.Errorf("reading webSocket message failed, err: %v", err)
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
						log.Errorf("ssh pty change windows size failed, err: %v", err)
					}
				}
			case wsMsgCmd:
				decodeBytes := wsData
				if err != nil {
					log.Errorf("websocket cmd string base64 decoding failed, err: %v", err)
				}
				if _, err := s.Session.Stdin.Write(decodeBytes); err != nil {
					log.Errorf("ws cmd bytes write to ssh.stdin pipe failed, err: %v", err)
				}
			}
		}
	}
}
func (s *SSHSession) SessionWait(quitChan chan bool) {
	if err := s.Session.Wait(); err != nil {
		log.Errorf("ssh session wait failed, err: %v", err)
		setQuit(quitChan)
	}
}

func setQuit(ch chan bool) {
	ch <- true
}
