package websocket

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/ssbeatty/oms/internal/ssh"
	"github.com/ssbeatty/oms/pkg/logger"
	"github.com/ssbeatty/oms/pkg/transport"
	"sync"
)

const (
	minSizeOfResizeMsg = 12
	defaultBufferSize  = 4096
)

var (
	// ZModemSZStart = []byte{13, 42, 42, 24, 66, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 13, 138, 17}
	ZModemSZStart = []byte{42, 42, 24, 66, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 13, 138, 17}
	// ZModemSZEnd = []byte{13, 42, 42, 24, 66, 48, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 50, 100, 13, 138}
	ZModemSZEnd   = []byte{42, 42, 24, 66, 48, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 50, 100, 13, 138}
	ZModemSZEndOO = []byte{79, 79}

	ZModemRZStart   = []byte{42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 48, 50, 51, 98, 101, 53, 48, 13, 138, 17}
	ZModemRZEStart  = []byte{42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 48, 54, 51, 102, 54, 57, 52, 13, 138, 17}
	ZModemRZSStart  = []byte{42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 50, 50, 51, 100, 56, 51, 50, 13, 138, 17}
	ZModemRZESStart = []byte{42, 42, 24, 66, 48, 49, 48, 48, 48, 48, 48, 50, 54, 51, 57, 48, 102, 54, 13, 138, 17}
	ZModemRZEnd     = []byte{42, 42, 24, 66, 48, 56, 48, 48, 48, 48, 48, 48, 48, 48, 48, 50, 50, 100, 13, 138}

	ZModemRZCtrlStart = []byte{42, 42, 24, 66, 48}
	ZModemRZCtrlEnd1  = []byte{13, 138, 17}
	ZModemRZCtrlEnd2  = []byte{13, 138}

	ZModemCancel = []byte{24, 24, 24, 24, 24, 8, 8, 8, 8, 8}
)

type message struct {
	Type string `json:"type"`
	Data []byte `json:"data"`
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

// SSHSession connect to ssh server using ssh session.
type SSHSession struct {
	*transport.Session
	once                           sync.Once
	logger                         *logger.Logger
	comboOutput                    *wsBufferWriter
	ZModemSZ, ZModemRZ, ZModemSZOO bool
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

// ReceiveWsMsg  receive websocket msg do some handling then write into ssh.session.stdin
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

func ByteContains(x, y []byte) bool {
	index := bytes.Index(x, y)
	if index == -1 {
		return false
	}
	return true
}

func (s *SSHSession) SendComboOutput(wsConn *websocket.Conn, exitCh chan struct{}) {
	//tells other go routine quit
	defer s.setQuit(exitCh)

	copyToMessage := func(r *wsBufferWriter, conn *websocket.Conn) {
		for {
			select {
			case <-exitCh:
				return
			default:
				var n int
				if n = r.buffer.Len(); n == 0 {
					continue
				}

				buff := r.buffer.Bytes()

				if s.ZModemSZOO {
					s.ZModemSZOO = false
					// 经过测试 centos7-8 使用的 lrzsz-0.12.20 在 sz 结束时会发送 ZModemSZEndOO
					// 而 deepin20 等自带更新的 lrzsz-0.12.21rc 在 sz 结束时不会发送 ZModemSZEndOO， 而前端 zmodemjs
					// 库只有接收到 ZModemSZEndOO 才会认为 sz 结束，固这里需判断 sz 结束时是否发送了 ZModemSZEndOO，
					// 如果没有则手动发送一个，以便保证前端 zmodemjs 库正常运行（如果不发送，会导致使用 sz 命令时无法连续
					// 下载多个文件）。
					if n < 2 {
						// 手动发送 ZModemSZEndOO
						conn.WriteMessage(websocket.BinaryMessage, ZModemSZEndOO)
					} else if n == 2 {
						if buff[0] == ZModemSZEndOO[0] && buff[1] == ZModemSZEndOO[1] {
							conn.WriteMessage(websocket.BinaryMessage, ZModemSZEndOO)
						} else {
							// 手动发送 ZModemSZEndOO
							conn.WriteMessage(websocket.BinaryMessage, ZModemSZEndOO)
						}
					} else {
						if buff[0] == ZModemSZEndOO[0] && buff[1] == ZModemSZEndOO[1] {
							conn.WriteMessage(websocket.BinaryMessage, buff[:2])
						} else {
							// 手动发送 ZModemSZEndOO
							conn.WriteMessage(websocket.BinaryMessage, ZModemSZEndOO)
						}
					}
				} else {
					if s.ZModemSZ {
						if uint32(n) == defaultBufferSize {
							// 如果读取的长度为 buffsize，则认为是在传输数据，
							// 这样可以提高 sz 下载速率，很低概率会误判 zmodem 取消操作
							conn.WriteMessage(websocket.BinaryMessage, buff[:n])
						} else {
							if ok := ByteContains(buff[:n], ZModemSZEnd); ok {
								s.ZModemSZ = false
								s.ZModemSZOO = true
								conn.WriteMessage(websocket.BinaryMessage, ZModemSZEnd)
							} else if ok := ByteContains(buff[:n], ZModemCancel); ok {
								s.ZModemSZ = false
								conn.WriteMessage(websocket.BinaryMessage, buff[:n])
							} else {
								conn.WriteMessage(websocket.BinaryMessage, buff[:n])
							}
						}
					} else if s.ZModemRZ {
						if ok := ByteContains(buff[:n], ZModemRZEnd); ok {
							s.ZModemRZ = false
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZEnd)
						} else if ok := ByteContains(buff[:n], ZModemCancel); ok {
							s.ZModemRZ = false
							conn.WriteMessage(websocket.BinaryMessage, buff[:n])
						} else {
							// rz 上传过程中服务器端还是会给客户端发送一些信息，比如心跳
							//conn.WriteJSON(&message{Type: messageTypeConsole, Data: buff[:n]})
							//conn.WriteMessage(websocket.BinaryMessage, buff[:n])

							startIndex := bytes.Index(buff[:n], ZModemRZCtrlStart)
							if startIndex != -1 {
								endIndex := bytes.Index(buff[:n], ZModemRZCtrlEnd1)
								if endIndex != -1 {
									ctrl := append(ZModemRZCtrlStart, buff[startIndex+len(ZModemRZCtrlStart):endIndex]...)
									ctrl = append(ctrl, ZModemRZCtrlEnd1...)
									conn.WriteMessage(websocket.BinaryMessage, ctrl)
								} else {
									endIndex = bytes.Index(buff[:n], ZModemRZCtrlEnd2)
									if endIndex != -1 {
										ctrl := append(ZModemRZCtrlStart, buff[startIndex+len(ZModemRZCtrlStart):endIndex]...)
										ctrl = append(ctrl, ZModemRZCtrlEnd2...)
										conn.WriteMessage(websocket.BinaryMessage, ctrl)
									}
								}
							}
						}
					} else {
						if ok := ByteContains(buff[:n], ZModemSZStart); ok {
							s.ZModemSZ = true
							conn.WriteMessage(websocket.BinaryMessage, ZModemSZStart)
						} else if ok = ByteContains(buff[:n], ZModemRZStart); ok {
							s.ZModemRZ = true
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZStart)
						} else if ok = ByteContains(buff[:n], ZModemRZEStart); ok {
							s.ZModemRZ = true
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZEStart)
						} else if ok = ByteContains(buff[:n], ZModemRZSStart); ok {
							s.ZModemRZ = true
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZSStart)
						} else if ok = ByteContains(buff[:n], ZModemRZESStart); ok {
							s.ZModemRZ = true
							conn.WriteMessage(websocket.BinaryMessage, ZModemRZESStart)
						} else {
							conn.WriteMessage(websocket.BinaryMessage, buff)
							conn.WriteJSON(&message{Type: "data", Data: buff})
						}
					}
				}
				r.buffer.Reset()
			}
		}
	}

	copyToMessage(s.comboOutput, wsConn)

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
