package models

import (
	"encoding/json"
	"github.com/astaxie/beego/orm"
	"github.com/pkg/sftp"
	"mime/multipart"
	"oms/logger"
	"oms/ssh"
	"strings"
	"time"
)

type Result struct {
	Status   bool
	HostId   int
	HostName string
	Msg      string
}

type FileInfo struct {
	Name    string
	ModTime time.Time
	Size    int64
	IsDir   bool
}

type ExportData struct {
	Tags   []*Tag
	Groups []*Group
	Hosts  []*Host
}

func ParseHostList(pType string, id int) []*Host {
	var hosts []*Host
	var o = orm.NewOrm()
	if pType == "host" {
		host := Host{Id: id}
		err := o.Read(&host)
		if err != nil {
			logger.Logger.Println(err)
		}
		hosts = append(hosts, &host)
	} else if pType == "tag" {
		host := new(Host)
		_, err := o.QueryTable(host).Filter("Tags__Tag__Id", id).All(&hosts)
		if err != nil {
			logger.Logger.Println(err)
		}
	} else {
		group := Group{Id: id}
		err := o.Read(&group)
		if err != nil {
			logger.Logger.Println(err)
		}
		if group.Mode == 0 {
			host := new(Host)
			_, err = o.QueryTable(host).Filter("Group__Id", id).All(&hosts)
			if err != nil {
				logger.Logger.Println(err)
			}
		} else {
			args := strings.Split(group.Params, " ")
			switch args[0] {
			case "-G":
				hosts := GetHostByGlob(args[1])
				return hosts
			case "-L":
				var hosts []*Host
				addrArgs := strings.Split(args[1], ",")
				for _, addr := range addrArgs {
					host := GetHostByAddr(addr)
					hosts = append(hosts, host...)
				}
				return hosts
			case "-E":
				hosts := GetHostByReg(args[1])
				return hosts
			default:
				hosts := GetHostByGlob(args[0])
				return hosts
			}
		}

	}
	return hosts
}

func RunCmdOne(host *Host, cmd string) ([]byte, error) {
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
	if err != nil {
		return nil, err
	}
	return client.Output(cmd)
}

func RunCmdOneAsync(host *Host, cmd string, ch chan *Result) {
	var result *Result
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	msg, err := client.Output(cmd)
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
	} else {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: string(msg)}
	}

	ch <- result
}

func RunCmd(hosts []*Host, cmd string) []*Result {
	var results []*Result
	channel := make(chan *Result, 20)
	for _, host := range hosts {
		go RunCmdOneAsync(host, cmd, channel)
	}
	for _, _ = range hosts {
		results = append(results, <-channel)
	}
	return results
}

func UploadFile(hosts []*Host, files []*multipart.FileHeader, remotePath string) []*Result {
	var results []*Result
	channel := make(chan *Result, 20)
	for _, host := range hosts {
		go UploadFileOneAsync(host, remotePath, files, channel)
	}
	for _, _ = range hosts {
		results = append(results, <-channel)
	}
	return results
}

func UploadFileOneAsync(host *Host, remote string, files []*multipart.FileHeader, ch chan *Result) {
	var result *Result
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
	if err != nil {
		result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		ch <- result
		return
	}
	// do upload
	for i, _ := range files {
		err = client.UploadFileOne(files[i], remote)
		if err != nil {
			logger.Logger.Println(err)
			result = &Result{HostId: host.Id, HostName: host.Name, Status: false, Msg: err.Error()}
		} else {
			result = &Result{HostId: host.Id, HostName: host.Name, Status: true, Msg: "success"}
		}
	}

	ch <- result
}

func GetStatus(host *Host) bool {
	var o = orm.NewOrm()
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
	if err != nil {
		host.Status = false
		_, err = o.Update(host)
		return false
	}
	session, err := client.SSHClient.NewSession()
	if err != nil {
		host.Status = false
		_, err = o.Update(host)
		return false
	}
	defer session.Close()
	host.Status = true
	_, err = o.Update(host)
	return true
}

func GetPathInfo(hostId int, path string) []*FileInfo {
	var results []*FileInfo
	host := GetHostById(hostId)
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
	if err != nil {
		return results
	}
	infos, err := client.ReadDir(path)
	if err != nil {
		return results
	}
	for i, _ := range infos {
		info := FileInfo{Name: infos[i].Name(), Size: infos[i].Size(), ModTime: infos[i].ModTime(), IsDir: infos[i].IsDir()}
		results = append(results, &info)
	}
	return results
}

func DownloadFile(hostId int, path string) *sftp.File {
	host := GetHostById(hostId)
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
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
	host := GetHostById(hostId)
	client, err := ssh.NewClient(host.Addr, host.Port, host.User, host.PassWord, host.KeyFile)
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
	groups := GetAllGroup()
	tags := GetAllTag()
	hosts := GetAllHost()
	data.Tags = append(data.Tags, tags...)
	data.Groups = append(data.Groups, groups...)
	data.Hosts = append(data.Hosts, hosts...)
	marshal, err := json.Marshal(data)
	if err != nil {
		logger.Logger.Println(err)
		return []byte{}, err
	}
	return marshal, nil
}

func ImportDbData(marshal []byte) error {
	data := &ExportData{}
	err := json.Unmarshal(marshal, &data)
	if err != nil {
		logger.Logger.Println(err)
		return err
	}
	for index, _ := range data.Tags {
		tag := data.Tags[index]
		ok := ExistedTag(tag.Name)
		if !ok {
			logger.Logger.Printf("Insert Tag %s", tag.Name)
			InsertTag(tag.Name)
		}
	}
	for index, _ := range data.Groups {
		group := data.Groups[index]
		ok := ExistedGroup(group.Name)
		if !ok {
			logger.Logger.Printf("Insert Group %s", group.Name)
			InsertGroup(group.Name, group.Params, group.Mode)
		}
	}
	for index, _ := range data.Hosts {
		host := data.Hosts[index]
		ok := ExistedHost(host.Name, host.Addr)
		if !ok {
			logger.Logger.Printf("Insert Host %s", host.Name)
			tags := make([]string, 0)
			for i, _ := range host.Tags {
				tags = append(tags, string(host.Tags[i].Id))
			}
			InsertHost(host.Name, host.User, host.Addr, host.Port, host.PassWord, host.Group.Id, tags, host.KeyFile)
		}
	}
	return nil
}
