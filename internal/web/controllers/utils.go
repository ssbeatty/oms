package controllers

import (
	"context"
	"fmt"
	"github.com/pkg/sftp"
	"io/fs"
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/internal/web/websocket"
	"oms/pkg/transport"
	"oms/pkg/utils"
	"os"
	"path"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

type FileInfo struct {
	Id            string    `json:"id"`
	Name          string    `json:"name"`
	ModTime       time.Time `json:"modDate"`
	Size          int64     `json:"size"`
	IsDir         bool      `json:"isDir,omitempty"`
	IsSymlink     bool      `json:"isSymlink,omitempty"`
	IsHidden      bool      `json:"isHidden,omitempty"`
	Ext           string    `json:"ext"`
	ChildrenCount int       `json:"childrenCount,omitempty"`
	ParentId      string    `json:"parentId,omitempty"`
	ChildrenIds   []string  `json:"children_ids,omitempty"`
	Icon          string    `json:"icon,omitempty"`
}

type FilePath struct {
	Files        []*FileInfo `json:"files"`
	FolderChains []*FileInfo `json:"folderChains"`
}

func (s *Service) GetSSHManager() *ssh.Manager {
	return s.sshManager
}

// RunCmdOneAsync 搭配RunCmd使用
func (s *Service) RunCmdOneAsync(host *models.Host, cmd string, sudo bool, ch chan *ssh.Result, wg *sync.WaitGroup) {
	var msg []byte
	var result *ssh.Result
	client, err := s.sshManager.NewClient(host)
	if err != nil {
		ch <- &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error(), Addr: host.Addr}
		return
	}
	session, err := client.NewPty()
	if err != nil {
		s.Logger.Errorf("RunCmdOneAsync create new session failed, err: %v", err)
		ch <- &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error(), Addr: host.Addr}
		return
	}
	defer session.Close()

	if sudo && client.GetTargetMachineOs() != transport.GOOSWindows {
		msg, err = session.Sudo(cmd, host.PassWord)
	} else {
		msg, err = session.Output(cmd)
	}
	if err != nil {
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: string(msg), Addr: host.Addr}
	} else {
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: string(msg), Addr: host.Addr}
	}

	ch <- result
	wg.Done()
}

// RunCmdExec 用于http接口
func (s *Service) RunCmdExec(hosts []*models.Host, cmd string, sudo bool) []*ssh.Result {
	var results []*ssh.Result
	channel := make(chan *ssh.Result, len(hosts))
	defer close(channel)
	wg := sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)
		go s.RunCmdOneAsync(host, cmd, sudo, channel, &wg)
	}
	wg.Wait()
	for i := 0; i < len(hosts); i++ {
		results = append(results, <-channel)
	}
	return results
}

func (s *Service) UploadFileStream(hosts []*models.Host, tmp *ssh.TempFile, remotePath string, ctx context.Context) {
	// 引用计数
	atomic.AddInt32(&tmp.Num, int32(len(hosts)))

	for _, host := range hosts {
		go func(host *models.Host) {
			client, err := s.sshManager.NewClientWithSftp(host)
			if err != nil {
				return
			}
			addr := fmt.Sprintf("%s:%d", host.Addr, host.Port)
			s.sshManager.UploadFileStream(client, remotePath, addr, tmp, ctx)
		}(host)
	}
}

func (s *Service) GetPathInfoExec(hostId int, p string) (*FilePath, error) {
	filePath := &FilePath{}
	var results []*FileInfo
	var folderChains []*FileInfo

	host, err := models.GetHostById(hostId)
	if err != nil {
		return nil, err
	}
	client, err := s.sshManager.NewClientWithSftp(host)
	if err != nil {
		return nil, err
	}
	switch p {
	case ".", "..":
		p, err = client.RealPath(p)
		if err != nil {
			s.Logger.Errorf("can not parse real path: %s, err: %v", p, err)
			return nil, err
		}
	case "", "~":
		p, err = client.RealPath(".")
		if err != nil {
			s.Logger.Errorf("can not parse real path: %s, err: %v", p, err)
			return nil, err
		}
	}
	infos, err := client.ReadDir(p)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(infos); i++ {
		var isDir, isSymlink bool
		var newHead os.FileInfo
		fId := filepath.ToSlash(filepath.Join(p, infos[i].Name()))
		if (infos[i].Mode() & fs.ModeType) == fs.ModeSymlink {
			newHead, err = client.Stat(fId)
			if err != nil {
				s.Logger.Errorf("GetPathInfoExec error when stat file: %s, err: %v", filepath.Join(p, infos[i].Name()), err)
				continue
			}
			isSymlink = true
			isDir = newHead.IsDir()
		} else {
			isDir = infos[i].IsDir()
		}
		info := FileInfo{
			Id:        fId,
			Name:      infos[i].Name(),
			Size:      infos[i].Size(),
			ModTime:   infos[i].ModTime(),
			IsDir:     isDir,
			IsSymlink: isSymlink,
			Ext:       utils.GetFileExt(infos[i].Name()),
			ParentId:  p,
			Icon:      utils.GetFileIcon(infos[i].Name(), isDir),
		}
		results = append(results, &info)
	}

	for {
		folderChains = append(folderChains, &FileInfo{
			Id:    p,
			Name:  path.Base(p),
			IsDir: true,
		})
		if p == path.Dir(p) {
			break
		} else {
			p = path.Dir(p)
		}
	}
	// Reverse
	for i, j := 0, len(folderChains)-1; i < j; i, j = i+1, j-1 {
		folderChains[i], folderChains[j] = folderChains[j], folderChains[i]
	}

	filePath.Files = results
	filePath.FolderChains = folderChains
	return filePath, nil
}

func (s *Service) DownloadFile(hostId int, path string) *sftp.File {
	host, err := models.GetHostById(hostId)
	if err != nil {
		return nil
	}
	client, err := s.sshManager.NewClientWithSftp(host)
	if err != nil {
		return nil
	}
	file, err := client.GetFile(path)
	if err != nil {
		return nil
	}
	return file
}

func (s *Service) DeleteFileOrDir(hostId int, path string) error {
	host, err := models.GetHostById(hostId)
	if err != nil {
		return err
	}
	client, err := s.sshManager.NewClientWithSftp(host)
	if err != nil {
		return err
	}
	if client.IsDir(path) {
		err = client.RemoveDir(path)
		if err != nil {
			return err
		}
	} else {
		err = client.Remove(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) MakeDir(hostId int, p, dir string) error {
	host, err := models.GetHostById(hostId)
	if err != nil {
		return err
	}
	client, err := s.sshManager.NewClientWithSftp(host)
	if err != nil {
		return err
	}
	switch p {
	case ".", "..":
		p, err = client.RealPath(p)
		if err != nil {
			s.Logger.Errorf("can not parse real path: %s, err: %v", p, err)
			return nil
		}
	case "", "~":
		p, err = client.RealPath(".")
		if err != nil {
			s.Logger.Errorf("can not parse real path: %s, err: %v", p, err)
			return nil
		}
	}
	realPath := filepath.ToSlash(filepath.Join(p, dir))
	return client.MkdirAll(realPath)
}

func (s *Service) runCmdWithContext(host *models.Host, cmd ssh.Command, ch chan *ssh.Result, ctx context.Context) {
	var (
		msg    []byte
		result *ssh.Result
	)

	client, err := s.sshManager.NewClientWithSftp(host)
	if err != nil {
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error(), Addr: host.Addr}
		ch <- result
		return
	}
	session, err := client.NewSessionWithPty(cmd.WindowSize.Cols, cmd.WindowSize.Rows)
	if err != nil {
		s.Logger.Errorf("RunCmdWithContext error when create new session failed, err: %v", err)
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error(), Addr: host.Addr}
		ch <- result
		return
	}

	if cmd.Sudo && client.GetTargetMachineOs() != transport.GOOSWindows {
		msg, err = session.SudoContext(ctx, cmd.Params, host.PassWord)
	} else {
		msg, err = session.OutputContext(ctx, cmd.Params)
	}

	if err != nil {
		// ctx 超时返回ctx.Err
		if ctx.Err() != nil {
			result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: ctx.Err().Error(), Addr: host.Addr}
		} else {
			result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: string(msg), Addr: host.Addr}
		}
	} else {
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: string(msg), Addr: host.Addr}
	}

	ch <- result
}

func (s *Service) runPlayerWithContext(host *models.Host, cmd ssh.Command, ch chan *ssh.Result, ctx context.Context) {
	var (
		msg    []byte
		result *ssh.Result
	)

	client, err := s.sshManager.NewClient(host)
	if err != nil {
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error(), Addr: host.Addr}
		ch <- result
		return
	}

	steps, err := s.sshManager.ParseSteps(cmd.Params)
	if err != nil {
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error(), Addr: host.Addr}
		ch <- result
		return
	}

	player := ssh.NewPlayer(client, steps, cmd.Sudo, &cmd.WindowSize)

	msg, err = player.Run(ctx)

	if err != nil {
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: string(msg), Addr: host.Addr}
	} else {
		result = &ssh.Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: string(msg), Addr: host.Addr}
	}

	ch <- result
}

// RunCmdWithContext 使用在websocket接口上
func (s *Service) RunCmdWithContext(host *models.Host, cmd ssh.Command, ch chan *ssh.Result) {
	ctx, cancel := context.WithTimeout(context.Background(), websocket.DefaultSSHCMDTimeout)
	defer cancel()
	switch cmd.Type {
	case ssh.CMDTypeShell:
		s.runCmdWithContext(host, cmd, ch, ctx)
	case ssh.CMDTypePlayer:
		s.runPlayerWithContext(host, cmd, ch, ctx)
	}
}

// GetRWFile 获取可读写的 sftp.File
func (s *Service) GetRWFile(hostId int, path string) *sftp.File {
	host, err := models.GetHostById(hostId)
	if err != nil {
		return nil
	}
	client, err := s.sshManager.NewClientWithSftp(host)
	if err != nil {
		return nil
	}
	file, err := client.GetRWFile(path)
	if err != nil {
		return nil
	}
	return file
}
