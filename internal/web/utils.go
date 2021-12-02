package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"mime/multipart"
	"oms/internal/models"
	"oms/internal/ssh"
	"oms/internal/utils"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Result struct {
	Status   bool   `json:"status"`
	HostId   int    `json:"host_id"`
	HostName string `json:"hostname"`
	Msg      string `json:"msg"`
}

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
}

type FilePath struct {
	Files        []*FileInfo `json:"files"`
	FolderChains []*FileInfo `json:"folderChains"`
}

type ExportData struct {
	Tags   []*models.Tag
	Groups []*models.Group
	Hosts  []*models.Host
}

func (s *Service) ParseHostList(pType string, id int) []*models.Host {
	var hosts []*models.Host
	if pType == "host" {
		host, err := models.GetHostById(id)
		if err != nil {
			s.logger.Errorf("ParseHostList error when GetHostById, err: %v", err)
			return nil
		}
		hosts = append(hosts, host)
	} else if pType == "tag" {
		tag, err := models.GetTagById(id)
		if err != nil {
			s.logger.Errorf("ParseHostList error when GetTagById, err: %v", err)
			return nil
		}
		hosts, err = models.GetHostsByTag(tag)
		if err != nil {
			s.logger.Errorf("ParseHostList error when GetHostsByTag, err: %v", err)
			return nil
		}
	} else {
		group, err := models.GetGroupById(id)
		if err != nil {
			s.logger.Errorf("ParseHostList error when GetGroupById, err: %v", err)
			return nil
		}
		if group.Mode == 0 {
			hosts, err = models.GetHostsByGroup(group)
			if err != nil {
				s.logger.Errorf("ParseHostList error when GetHostsByGroup, err: %v", err)
				return nil
			}
		} else {
			args := strings.Split(group.Params, " ")
			if len(args) < 2 {
				log.Errorf("group params error, params: %s", group.Params)
				return nil
			} else {
				if strings.Contains(args[1], "\"") {
					args[1] = strings.ReplaceAll(args[1], "\"", "")
				}
			}
			switch args[0] {
			case "-G":
				hosts, err := models.GetHostByGlob(args[1])
				if err != nil {
					s.logger.Errorf("ParseHostList error when GetHostByGlob, err: %v", err)
					return nil
				}
				return hosts
			case "-L":
				var hosts []*models.Host
				addrArgs := strings.Split(args[1], ",")
				for _, addr := range addrArgs {
					host, err := models.GetHostByAddr(addr)
					if err != nil {
						s.logger.Errorf("ParseHostList error when GetHostByAddr, err: %v", err)
						return nil
					}
					hosts = append(hosts, host...)
				}
				return hosts
			case "-E":
				hosts, err := models.GetHostByReg(args[1])
				if err != nil {
					s.logger.Errorf("ParseHostList error when GetHostByReg, err: %v", err)
					return nil
				}
				return hosts
			default:
				hosts, err := models.GetHostByGlob(args[0])
				if err != nil {
					s.logger.Errorf("ParseHostList error when GetHostByGlob, err: %v", err)
					return nil
				}
				return hosts
			}
		}

	}
	return hosts
}

// RunCmdOneAsync 搭配RunCmd使用
func (s *Service) RunCmdOneAsync(host *models.Host, cmd string, sudo bool, ch chan *Result, wg *sync.WaitGroup) {
	var msg []byte
	var result *Result
	client, err := s.sshManager.NewClient(host)
	if err != nil {
		ch <- &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		return
	}
	session, err := client.NewPty()
	if err != nil {
		s.logger.Errorf("RunCmdOneAsync create new session failed, err: %v", err)
		ch <- &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		return
	}
	defer session.Close()

	if sudo {
		msg, err = session.Sudo(cmd, host.PassWord)
	} else {
		msg, err = session.Output(cmd)
	}
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: string(msg)}
	} else {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: string(msg)}
	}

	ch <- result
	wg.Done()
}

// RunCmdExec 用于http接口
func (s *Service) RunCmdExec(hosts []*models.Host, cmd string, sudo bool) []*Result {
	var results []*Result
	channel := make(chan *Result, len(hosts))
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

func (s *Service) UploadFileUnBlock(hosts []*models.Host, files []*multipart.FileHeader, remotePath string) {
	wg := sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)
		go s.UploadFileOneAsync(host, remotePath, files, &wg)
	}
	wg.Wait()
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

func (s *Service) UploadFileOneAsync(host *models.Host, remote string, files []*multipart.FileHeader, wg *sync.WaitGroup) {
	client, err := s.sshManager.NewClientWithSftp(host)
	if err != nil {
		return
	}
	// do upload
	for i := 0; i < len(files); i++ {
		addr := fmt.Sprintf("%s:%d", host.Addr, host.Port)
		go s.sshManager.UploadFileOneAsync(client, files[i], remote, addr, files[i].Filename)
	}
	wg.Done()
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
			s.logger.Errorf("can not parse real path: %s, err: %v", p, err)
			return nil, err
		}
	case "", "~":
		p, err = client.RealPath(".")
		if err != nil {
			s.logger.Errorf("can not parse real path: %s, err: %v", p, err)
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
				s.logger.Errorf("GetPathInfoExec error when stat file: %s, err: %v", filepath.Join(p, infos[i].Name()), err)
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
			s.logger.Errorf("can not parse real path: %s, err: %v", p, err)
			return nil
		}
	case "", "~":
		p, err = client.RealPath(".")
		if err != nil {
			s.logger.Errorf("can not parse real path: %s, err: %v", p, err)
			return nil
		}
	}
	realPath := filepath.ToSlash(filepath.Join(p, dir))
	return client.MkdirAll(realPath)
}

func (s *Service) ExportDbData() ([]byte, error) {
	data := &ExportData{}
	groups, _ := models.GetAllGroup()
	tags, _ := models.GetAllTag()
	hosts, _ := models.GetAllHost()
	data.Tags = append(data.Tags, tags...)
	data.Groups = append(data.Groups, groups...)
	data.Hosts = append(data.Hosts, hosts...)
	marshal, err := json.Marshal(data)
	if err != nil {
		s.logger.Error(err)
		return []byte{}, err
	}
	return marshal, nil
}

func (s *Service) ImportDbData(marshal []byte) error {
	data := &ExportData{}
	err := json.Unmarshal(marshal, &data)
	if err != nil {
		return err
	}
	for index := 0; index < len(data.Tags); index++ {
		tag := data.Tags[index]
		ok := models.ExistedTag(tag.Name)
		if !ok {
			s.logger.Infof("ImportDbData error when Insert Tag %s", tag.Name)
			_, _ = models.InsertTag(tag.Name)
		}
	}
	for index := 0; index < len(data.Groups); index++ {
		group := data.Groups[index]
		ok := models.ExistedGroup(group.Name)
		if !ok {
			s.logger.Errorf("ImportDbData error when Insert Group %s", group.Name)
			_, _ = models.InsertGroup(group.Name, group.Params, group.Mode)
		}
	}
	for index := 0; index < len(data.Hosts); index++ {
		host := data.Hosts[index]
		ok := models.ExistedHost(host.Name, host.Addr)
		if !ok {
			s.logger.Errorf("ImportDbData error when Insert Host %s", host.Name)
			tags := make([]int, 0)
			for i := 0; i < len(host.Tags); i++ {
				tags = append(tags, host.Tags[i].Id)
			}
			// todo
			_, _ = models.InsertHost(host.Name, host.User, host.Addr, host.Port, host.PassWord, host.GroupId, tags, 0)
		}
	}
	return nil
}

func (s *Service) runCmdWithContext(host *models.Host, cmd string, sudo bool, ch chan *Result, ctx context.Context) {
	var msg []byte
	var errMsg string
	var result *Result

	quit := make(chan bool, 1)
	defer close(quit)

	client, err := s.sshManager.NewClient(host)
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	session, err := client.NewPty()
	if err != nil {
		s.logger.Errorf("RunCmdWithContext error when create new session failed, err: %v", err)
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}

	go func() {
		defer session.Close()
		select {
		case <-ctx.Done():
			errMsg = "cmd timeout"
			return
		case <-quit:
			return
		}
	}()

	if sudo {
		msg, err = session.Sudo(cmd, host.PassWord)
	} else {
		msg, err = session.Output(cmd)
	}
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: string(msg)}
	} else {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: string(msg)}
	}

	if errMsg != "" {
		result.Msg = errMsg
	}
	ch <- result
}

// RunCmdWithContext 使用在websocket接口上
func (s *Service) RunCmdWithContext(host *models.Host, cmd string, sudo bool, ch chan *Result) {
	ctx, cancel := context.WithTimeout(context.Background(), SSHTimeDeadline)
	defer cancel()
	s.runCmdWithContext(host, cmd, sudo, ch, ctx)
}
