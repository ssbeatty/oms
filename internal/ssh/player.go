package ssh

import (
	"bytes"
	"context"
	"fmt"
	"oms/internal/ssh/buildin"
	"oms/pkg/transport"
)

type Player struct {
	sudo   bool
	client *transport.Client
	Steps  []buildin.Step `json:"steps"`
}

func NewPlayer(client *transport.Client, steps []buildin.Step, sudo bool) *Player {
	return &Player{
		sudo:   sudo,
		client: client,
		Steps:  steps,
	}
}

func (p *Player) Run(ctx context.Context) ([]byte, error) {
	var (
		err     error
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
		session, err = p.client.NewPty()
		if err != nil {
			return buf.Bytes(), err
		}

		// todo 优化样式
		buf.WriteString(fmt.Sprintf("[Step %8s] ==> %s\n", step.Name(), step.ID()))
		msg, err := step.Exec(session, p.sudo)

		buf.Write(msg)

		if err != nil {
			buf.Write([]byte(err.Error()))
			return buf.Bytes(), err
		}

		session.Close()
	}

	return buf.Bytes(), nil
}
