package wscontrol

import (
	"bytes"
	"github.com/gorilla/websocket"
	"log"
	"oms/transport"
	"sync"
)

// copy data from WebSocket to ssh server
// and copy data from ssh server to WebSocket

// write data to WebSocket
// the data comes from ssh server.
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

// connect to ssh server using ssh session.
type SshConn struct {
	Session *transport.Session
}

type WSConnect struct {
	*websocket.Conn
}

func (w *WSConnect) Write(p []byte) (int, error) {
	err := w.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

//flushComboOutput flush ssh.session combine output into websocket response
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

// setup ssh shell session
// set Session and StdinPipe here,
// and the Session.Stdout and Session.Sdterr are also set.
func NewSshConn(cols, rows int, sshClient *transport.Client) (*SshConn, error) {
	sshSession, err := sshClient.NewSessionWithPty(cols, rows)
	if err != nil {
		return nil, err
	}
	if err := sshSession.SSHSession.Shell(); err != nil {
		return nil, err
	}
	return &SshConn{Session: sshSession}, nil
}

func (s *SshConn) Close() {
	if s.Session != nil {
		s.Session.Close()
	}

}

//ReceiveWsMsg  receive websocket msg do some handling then write into ssh.session.stdin
func (ssConn *SshConn) ReceiveWsMsg(wsConn *websocket.Conn, logBuff *bytes.Buffer, exitCh chan bool) {
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
				log.Println("reading webSocket message failed")
				return
			}
			//unmashal bytes into struct
			msgObj := wsMsg{
				Type: "cmd",
				Cmd:  "",
				Rows: 40,
				Cols: 180,
			}
			//if err := json.Unmarshal(wsData, &msgObj); err != nil {
			//	logrus.WithError(err).WithField("wsData", string(wsData)).Error("unmarshal websocket message failed")
			//}
			switch msgObj.Type {
			case wsMsgCmd:
				//handle xterm.js stdin
				//decodeBytes, err := base64.StdEncoding.DecodeString(msgObj.Cmd)
				decodeBytes := wsData
				if err != nil {
					log.Println("websock cmd string base64 decoding failed")
				}
				if _, err := ssConn.Session.Stdin.Write(decodeBytes); err != nil {
					log.Println("ws cmd bytes write to ssh.stdin pipe failed")
				}
			}
		}
	}
}
func (ssConn *SshConn) SessionWait(quitChan chan bool) {
	if err := ssConn.Session.SSHSession.Wait(); err != nil {
		log.Println("ssh session wait failed")
		setQuit(quitChan)
	}
}

func setQuit(ch chan bool) {
	ch <- true
}
