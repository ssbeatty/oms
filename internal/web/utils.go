package web

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/sftp"
	"io/fs"
	"mime/multipart"
	"oms/internal/models"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Result struct {
	Status   bool   `json:"status"`
	HostId   int    `json:"host_id"`
	HostName string `json:"hostname"`
	Msg      string `json:"msg"`
}

type FileInfo struct {
	Name    string    `json:"name"`
	ModTime time.Time `json:"mod_time"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"is_dir"`
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
	client, err := s.sshManager.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	session, err := client.NewSession()
	if err != nil {
		s.logger.Errorf("RunCmdOneAsync create new session failed, err: %v", err)
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

func (s *Service) UploadFile(hosts []*models.Host, files []*multipart.FileHeader, remotePath string) []*Result {
	var results []*Result
	channel := make(chan *Result, len(hosts))
	defer close(channel)
	wg := sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)
		go s.UploadFileOneSync(host, remotePath, files, channel, &wg)
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

func (s *Service) UploadFileOneSync(host *models.Host, remote string, files []*multipart.FileHeader, ch chan *Result, wg *sync.WaitGroup) {
	var result *Result
	client, err := s.sshManager.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	// do upload
	for i := 0; i < len(files); i++ {
		err = client.UploadFileOne(files[i], remote)
		if err != nil {
			s.logger.Error(err)
			result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		} else {
			result = &Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: "success"}
		}
	}

	ch <- result
	wg.Done()
}

func (s *Service) UploadFileOneAsync(host *models.Host, remote string, files []*multipart.FileHeader, wg *sync.WaitGroup) {
	client, err := s.sshManager.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
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

func (s *Service) GetPathInfoExec(hostId int, path string) []*FileInfo {
	var results []*FileInfo
	host, _ := models.GetHostById(hostId)
	client, err := s.sshManager.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		return results
	}
	infos, err := client.ReadDir(path)
	if err != nil {
		return results
	}
	for i := 0; i < len(infos); i++ {
		var isDir bool
		var newHead os.FileInfo
		if (infos[i].Mode() & fs.ModeType) == fs.ModeSymlink {
			newHead, err = client.Stat(filepath.ToSlash(filepath.Join(path, infos[i].Name())))
			if err != nil {
				s.logger.Errorf("GetPathInfoExec error when stat file: %s, err: %v", filepath.Join(path, infos[i].Name()), err)
				continue
			}
			isDir = newHead.IsDir()
		} else {
			isDir = infos[i].IsDir()
		}
		info := FileInfo{Name: infos[i].Name(), Size: infos[i].Size(), ModTime: infos[i].ModTime(), IsDir: isDir}
		results = append(results, &info)
	}
	return results
}

func (s *Service) DownloadFile(hostId int, path string) *sftp.File {
	host, _ := models.GetHostById(hostId)
	client, err := s.sshManager.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
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
	host, _ := models.GetHostById(hostId)
	client, err := s.sshManager.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
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
			tags := make([]string, 0)
			for i := 0; i < len(host.Tags); i++ {
				tags = append(tags, strconv.Itoa(host.Tags[i].Id))
			}
			_, _ = models.InsertHost(host.Name, host.User, host.Addr, host.Port, host.PassWord, host.GroupId, tags, host.KeyFile)
		}
	}
	return nil
}

// RunCmdWithContext 使用在websocket接口上
func (s *Service) RunCmdWithContext(host *models.Host, cmd string, sudo bool, ch chan *Result, ctx context.Context) {
	var msg []byte
	var result *Result

	quit := make(chan bool, 1)
	defer close(quit)

	client, err := s.sshManager.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	session, err := client.NewSessionWithPty(20, 20)
	if err != nil {
		s.logger.Errorf("RunCmdWithContext error when create new session failed, err: %v", err)
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}

	go func() {
		select {
		case <-ctx.Done():
			_ = session.Close()
			result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: "command timeout!"}
			ch <- result
		case <-quit:
			_ = session.Close()
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

	ch <- result
}
