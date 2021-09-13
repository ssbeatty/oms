package transport

import "io"

/*
Transport 用来跟agent或者ssh通讯的接口
*/

type IClient interface {
	NewSession() (ISession, error)
}

type ISession interface {
	Run(cmd string) error
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.Reader, error)
	StderrPipe() (io.Reader, error)
	Close() error
}

type ITransport interface {
	Connect() (Client, error)
	DisConnect() error
}
