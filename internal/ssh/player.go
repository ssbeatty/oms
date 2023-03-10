package ssh

import (
	"bytes"
	"context"
	"fmt"
	"github.com/fatih/color"
	"github.com/ssbeatty/oms/pkg/transport"
	"github.com/ssbeatty/oms/pkg/types"
)

var (
	cyan = color.New(color.FgCyan).SprintFunc()
)

type Player struct {
	sudo   bool
	client *transport.Client
	Steps  []types.Step `json:"steps"`
	size   *WindowSize
}

func NewPlayer(client *transport.Client, steps []types.Step, sudo bool, size *WindowSize) *Player {
	return &Player{
		sudo:   sudo,
		client: client,
		Steps:  steps,
		size:   size,
	}
}

func (p *Player) Run(ctx context.Context) ([]byte, error) {
	var (
		buf     bytes.Buffer
		session *transport.Session
		quit    = make(chan struct{}, 1)
	)

	defer close(quit)

	go func() {
		select {
		case <-ctx.Done():
			if session != nil {
				session.Close()
			}
		case <-quit:
			return
		}
	}()

	for _, step := range p.Steps {
		var (
			err error
		)

		if p.size != nil {
			session, err = p.client.NewSessionWithPty(p.size.Cols, p.size.Rows)
		} else {
			session, err = p.client.NewPty()
		}
		if err != nil {
			buf.Write([]byte(err.Error()))
			return buf.Bytes(), err
		}

		buf.WriteString(cyan(fmt.Sprintf("[Step %8s] ==> \"%s\"\r\n", step.Name(), step.ID())))
		msg, err := step.Exec(session, p.sudo)

		buf.Write(msg)

		if err != nil {
			buf.Write([]byte(err.Error()))
		}

		session.Close()
	}

	return buf.Bytes(), nil
}
