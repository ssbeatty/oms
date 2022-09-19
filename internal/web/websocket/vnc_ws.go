package websocket

import (
	"github.com/gorilla/websocket"
	"net"
	"oms/pkg/logger"
	"sync"
)

type VNCForward struct {
	once     sync.Once
	logger   *logger.Logger
	quitChan chan struct{}
	wsConn   *websocket.Conn
	tcpConn  net.Conn
}

func NewVNCForward(wsConn *websocket.Conn, tcpConn net.Conn, logger *logger.Logger, quitChan chan struct{}) *VNCForward {
	return &VNCForward{
		once:     sync.Once{},
		logger:   logger,
		quitChan: quitChan,
		wsConn:   wsConn,
		tcpConn:  tcpConn,
	}
}

func (vf *VNCForward) Serve() {
	go vf.forwardTcp()
	go vf.forwardWeb()

	<-vf.quitChan
}

func (vf *VNCForward) Close() {
	if vf.tcpConn != nil {
		vf.tcpConn.Close()
	}
	if vf.wsConn != nil {
		vf.wsConn.Close()
	}
}

func (vf *VNCForward) forwardTcp() {
	var tcpBuffer [1024]byte
	defer func() {
		vf.setQuit()
		vf.logger.Debug("vnc forward tcp exit.")
	}()
	for {
		select {
		case <-vf.quitChan:
			return
		default:
			if (vf.tcpConn == nil) || (vf.wsConn == nil) {
				return
			}
			n, err := vf.tcpConn.Read(tcpBuffer[0:])
			if err != nil {
				vf.logger.Errorf("reading from TCP failed: %s", err)
				return
			} else {
				if err := vf.wsConn.WriteMessage(websocket.BinaryMessage, tcpBuffer[0:n]); err != nil {
					vf.logger.Errorf("writing to WS failed: %s", err)
					return
				}
			}
		}
	}
}

func (vf *VNCForward) forwardWeb() {
	defer func() {
		if err := recover(); err != nil {
			vf.logger.Errorf("reading from WS failed: %s", err)
		}
		vf.setQuit()
		vf.logger.Debug("vnc forward web exit.")
	}()
	for {
		select {
		case <-vf.quitChan:
			return
		default:
			if (vf.tcpConn == nil) || (vf.wsConn == nil) {
				return
			}

			_, buffer, err := vf.wsConn.ReadMessage()
			if err == nil {
				if _, err := vf.tcpConn.Write(buffer); err != nil {
					vf.logger.Errorf("writing to TCP failed: %s", err)
					return
				}
			} else {
				vf.logger.Errorf("read message from websocket failed: %s", err)
				return
			}
		}
	}
}

func (vf *VNCForward) setQuit() {
	vf.once.Do(func() {
		close(vf.quitChan)
	})
}
