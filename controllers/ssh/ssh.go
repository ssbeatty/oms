package ssh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Run Execute cmd on the remote host for daemon service
func (c *Client) Run(cmd string) {
	session, err := c.SSHClient.NewSession()
	if err != nil {
		return
	}
	defer session.Close()

	err = session.Start(cmd)
	if err != nil {
		fmt.Printf("exec command:%v error:%v\n", cmd, err)
	}
	fmt.Printf("Waiting for command:%v to finish...\n", cmd)
	//阻塞等待fork出的子进程执行的结果，和cmd.Start()配合使用[不等待回收资源，会导致fork出执行shell命令的子进程变为僵尸进程]
	err = session.Wait()
	if err != nil {
		fmt.Printf(":Command finished with error: %v\n", err)
	}
	return
}

//Exec Execute cmd on the remote host and bind stderr and stdout
func (c *Client) Exec1(cmd string) error {

	// New Session
	session, err := c.SSHClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// go func() {
	// 	time.Sleep(2419200 * time.Second)
	// 	conn.Close()
	// }()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	err = session.Run(cmd)
	session.Close()
	return nil

}

//Exec Execute cmd on the remote host and bind stderr and stdout
func (c *Client) Exec(cmd string) error {
	session, err := c.SSHClient.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	// session.Run(cmd)
	// return session.CombinedOutput(cmd)
	stdout, err := session.StdoutPipe()
	// stderr, err = session.StderrPipe()
	if err != nil {
		fmt.Println(err)
		return err
	}

	var b bytes.Buffer
	session.Stderr = &b
	session.Start(cmd)
	//创建一个流来读取管道内内容，这里逻辑是通过一行一行的读取的
	reader := bufio.NewReader(stdout)

	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		print(line)
	}

	//阻塞直到该命令执行完成，该命令必须是被Start方法开始执行的
	session.Wait()
	if b.Len() > 0 {
		return errors.New(b.String())
	}
	return nil
}

// Output Execute cmd on the remote host and return stderr and stdout
func (c *Client) Output(cmd string) ([]byte, error) {
	session, err := c.SSHClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	// session.Run(cmd)
	// return session.CombinedOutput(cmd)
	return session.Output(cmd)
}

// RunScript Executes a shell script file on the remote machine.
// It is copied in the tmp folder and ran in a single session.
// chmod +x is applied before running.
// Returns an SshResponse and an error if any has occured
func (c *Client) RunScript(scriptPath string) ([]byte, error) {
	session, err := c.SSHClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// 1. 上传 script
	remotePath := fmt.Sprintf("/tmp/script/%s", filepath.Base(scriptPath))
	if err := c.UploadFile(scriptPath, remotePath); err != nil {
		return nil, err
	}
	// 2. 执行script
	rCmd := fmt.Sprintf("chmod +x %s ; %s", remotePath, remotePath)
	return session.CombinedOutput(rCmd)
}
