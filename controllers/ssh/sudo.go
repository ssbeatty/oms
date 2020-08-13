package ssh

import (
	"bytes"
	"io"
	"sync"
)

// This is the phrase that tells us sudo is looking for a password via stdin
const sudoPwPrompt = "sudo_password"

// sudoWriter is used to both combine stdout and stderr as well as
// look for a password request from sudo.
type sudoWriter struct {
	b     bytes.Buffer
	pw    string    // The password to pass to sudo (if requested)
	stdin io.Writer // The writer from the ssh session
	m     sync.Mutex
}

func (w *sudoWriter) Write(p []byte) (int, error) {
	// If we get the sudo password prompt phrase send the password via stdin
	// and don't write it to the buffer.
	if string(p) == sudoPwPrompt {
		w.stdin.Write([]byte(w.pw + "\n"))
		w.pw = "" // We don't need the password anymore so reset the string
		return len(p), nil
	}

	w.m.Lock()
	defer w.m.Unlock()

	return w.b.Write(p)
}

// ExecSu Execute cmd via sudo. Do not include the sudo command in
// the cmd string. For example: Client.ExecSudo("uptime", "password").
// If you are using passwordless sudo you can use the regular Exec()
// function.
func (c *Client) ExecSu(cmd, passwd string) ([]byte, error) {
	session, err := c.SSHClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// -n run non interactively
	// -p specify the prompt. We do this to know that sudo is asking for a passwd
	// -S Writes the prompt to StdErr and reads the password from StdIn
	cmd = "sudo -p " + sudoPwPrompt + " -S " + cmd

	// Use the sudoRW struct to handle the interaction with sudo and capture the
	// output of the command
	w := &sudoWriter{
		pw: passwd,
	}
	w.stdin, err = session.StdinPipe()
	if err != nil {
		return nil, err
	}

	// Combine stdout, stderr to the same writer which also looks for the sudo
	// password prompt
	session.Stdout = w
	session.Stderr = w

	err = session.Run(cmd)

	return w.b.Bytes(), err
}
