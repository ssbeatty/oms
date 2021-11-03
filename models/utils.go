package models

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"mime/multipart"
	"oms/pkg/transport"
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
	Tags   []*Tag
	Groups []*Group
	Hosts  []*Host
}

func ParseHostList(pType string, id int) []*Host {
	var hosts []*Host
	if pType == "host" {
		host, err := GetHostById(id)
		if err != nil {
			log.Errorf("ParseHostList error when GetHostById, err: %v", err)
			return nil
		}
		hosts = append(hosts, host)
	} else if pType == "tag" {
		tag, err := GetTagById(id)
		if err != nil {
			log.Errorf("ParseHostList error when GetTagById, err: %v", err)
			return nil
		}
		hosts, err = GetHostsByTag(tag)
		if err != nil {
			log.Errorf("ParseHostList error when GetHostsByTag, err: %v", err)
			return nil
		}
	} else {
		group, err := GetGroupById(id)
		if err != nil {
			log.Errorf("ParseHostList error when GetGroupById, err: %v", err)
			return nil
		}
		if group.Mode == 0 {
			hosts, err = GetHostsByGroup(group)
			if err != nil {
				log.Errorf("ParseHostList error when GetHostsByGroup, err: %v", err)
				return nil
			}
		} else {
			args := strings.Split(group.Params, " ")
			switch args[0] {
			case "-G":
				hosts, err := GetHostByGlob(args[1])
				if err != nil {
					log.Errorf("ParseHostList error when GetHostByGlob, err: %v", err)
					return nil
				}
				return hosts
			case "-L":
				var hosts []*Host
				addrArgs := strings.Split(args[1], ",")
				for _, addr := range addrArgs {
					host, err := GetHostByAddr(addr)
					if err != nil {
						log.Errorf("ParseHostList error when GetHostByAddr, err: %v", err)
						return nil
					}
					hosts = append(hosts, host...)
				}
				return hosts
			case "-E":
				hosts, err := GetHostByReg(args[1])
				if err != nil {
					log.Errorf("ParseHostList error when GetHostByReg, err: %v", err)
					return nil
				}
				return hosts
			default:
				hosts, err := GetHostByGlob(args[0])
				if err != nil {
					log.Errorf("ParseHostList error when GetHostByGlob, err: %v", err)
					return nil
				}
				return hosts
			}
		}

	}
	return hosts
}

// RunCmdOneAsync 搭配RunCmd使用
func RunCmdOneAsync(host *Host, cmd string, sudo bool, ch chan *Result, wg *sync.WaitGroup) {
	var msg []byte
	var result *Result
	client, err := transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	session, err := client.NewSession()
	if err != nil {
		log.Errorf("create new session failed, err: %v", err)
	}
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

// RunCmd 用于http接口
func RunCmd(hosts []*Host, cmd string, sudo bool) []*Result {
	var results []*Result
	channel := make(chan *Result, len(hosts))
	defer close(channel)
	wg := sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)
		go RunCmdOneAsync(host, cmd, sudo, channel, &wg)
	}
	wg.Wait()
	for i := 0; i < len(hosts); i++ {
		results = append(results, <-channel)
	}
	return results
}

func UploadFile(hosts []*Host, files []*multipart.FileHeader, remotePath string) []*Result {
	var results []*Result
	channel := make(chan *Result, len(hosts))
	defer close(channel)
	wg := sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)
		go UploadFileOneSync(host, remotePath, files, channel, &wg)
	}
	wg.Wait()
	for i := 0; i < len(hosts); i++ {
		results = append(results, <-channel)
	}
	return results
}

func UploadFileUnBlock(hosts []*Host, files []*multipart.FileHeader, remotePath string) {
	wg := sync.WaitGroup{}
	for _, host := range hosts {
		wg.Add(1)
		go UploadFileOneAsync(host, remotePath, files, &wg)
	}
	wg.Wait()
}

func UploadFileOneSync(host *Host, remote string, files []*multipart.FileHeader, ch chan *Result, wg *sync.WaitGroup) {
	var result *Result
	client, err := transport.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	// do upload
	for i := 0; i < len(files); i++ {
		err = client.UploadFileOne(files[i], remote)
		if err != nil {
			log.Println(err)
			result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		} else {
			result = &Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: "success"}
		}
	}

	ch <- result
	wg.Done()
}

func UploadFileOneAsync(host *Host, remote string, files []*multipart.FileHeader, wg *sync.WaitGroup) {
	client, err := transport.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		return
	}
	// do upload
	for i := 0; i < len(files); i++ {
		addr := fmt.Sprintf("%s:%d", host.Addr, host.Port)
		client.UploadFileOneAsync(files[i], remote, addr, files[i].Filename)
	}
	wg.Done()
}

func GetStatus(host *Host) bool {
	client, err := transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		host.Status = false
		_ = UpdateHostStatus(host)
		return false
	}
	session, err := client.NewSession()
	if err != nil {
		host.Status = false
		_ = UpdateHostStatus(host)
		return false
	}
	defer session.Close()
	host.Status = true
	_ = UpdateHostStatus(host)
	return true
}

func GetPathInfo(hostId int, path string) []*FileInfo {
	var results []*FileInfo
	host, _ := GetHostById(hostId)
	client, err := transport.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
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
			newHead, _ = client.Stat(filepath.ToSlash(filepath.Join(path, infos[i].Name())))
			isDir = newHead.IsDir()
		} else {
			isDir = infos[i].IsDir()
		}
		info := FileInfo{Name: infos[i].Name(), Size: infos[i].Size(), ModTime: infos[i].ModTime(), IsDir: isDir}
		results = append(results, &info)
	}
	return results
}

func DownloadFile(hostId int, path string) *sftp.File {
	host, _ := GetHostById(hostId)
	client, err := transport.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		return nil
	}
	file, err := client.GetFile(path)
	if err != nil {
		return nil
	}
	return file
}

func DeleteFileOrDir(hostId int, path string) error {
	host, _ := GetHostById(hostId)
	client, err := transport.NewClientWithSftp(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
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

func ExportDbData() ([]byte, error) {
	data := &ExportData{}
	groups, _ := GetAllGroup()
	tags, _ := GetAllTag()
	hosts, _ := GetAllHost()
	data.Tags = append(data.Tags, tags...)
	data.Groups = append(data.Groups, groups...)
	data.Hosts = append(data.Hosts, hosts...)
	marshal, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return []byte{}, err
	}
	return marshal, nil
}

func ImportDbData(marshal []byte) error {
	data := &ExportData{}
	err := json.Unmarshal(marshal, &data)
	if err != nil {
		log.Println(err)
		return err
	}
	for index := 0; index < len(data.Tags); index++ {
		tag := data.Tags[index]
		ok := ExistedTag(tag.Name)
		if !ok {
			log.Printf("Insert Tag %s", tag.Name)
			_, _ = InsertTag(tag.Name)
		}
	}
	for index := 0; index < len(data.Groups); index++ {
		group := data.Groups[index]
		ok := ExistedGroup(group.Name)
		if !ok {
			log.Printf("Insert Group %s", group.Name)
			_, _ = InsertGroup(group.Name, group.Params, group.Mode)
		}
	}
	for index := 0; index < len(data.Hosts); index++ {
		host := data.Hosts[index]
		ok := ExistedHost(host.Name, host.Addr)
		if !ok {
			log.Printf("Insert Host %s", host.Name)
			tags := make([]string, 0)
			for i := 0; i < len(host.Tags); i++ {
				tags = append(tags, strconv.Itoa(host.Tags[i].Id))
			}
			_, _ = InsertHost(host.Name, host.User, host.Addr, host.Port, host.PassWord, host.GroupId, tags, host.KeyFile)
		}
	}
	return nil
}

// RunCmdWithContext 使用在websocket接口上
func RunCmdWithContext(host *Host, cmd string, sudo bool, ch chan *Result, ctx context.Context) {
	var msg []byte
	var result *Result

	quit := make(chan bool, 1)
	defer close(quit)

	client, err := transport.NewClient(host.Addr, host.Port, host.User, host.PassWord, []byte(host.KeyFile))
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	session, err := client.NewSessionWithPty(20, 20)
	if err != nil {
		log.Errorf("create new session failed, err: %v", err)
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
