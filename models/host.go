package models

import (
	"log"
	"regexp"
	"strconv"
	"strings"
)

// Model Struct
type Host struct {
	Id       int
	Name     string `gorm:"size:100;not null"`
	User     string `gorm:"size:128;not null"`
	Addr     string `gorm:"size:128;not null"`
	Port     int    `gorm:"default:22"`
	PassWord string `gorm:"size:128;not null"`
	KeyFile  string `gorm:"type:text"`

	Status bool `gorm:"default:false"`

	GroupId int
	Group   Group
	Tags    []Tag `gorm:"many2many:host_tag"`
}

func GetHostById(id int) *Host {
	host := Host{}
	result := db.Where("id = ?", id).Preload("Tags").Preload("Group").First(&host)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return &host
}

func ExistedHost(name string, addr string) bool {
	var hosts []*Host
	result := db.Where(&Host{Name: name, Addr: addr}).Find(&hosts)
	if result.Error != nil {
		log.Println(result.Error)
		return false
	}
	if len(hosts) == 0 {
		return false
	}
	return true
}

func DeleteHostById(id int) bool {
	host := Host{}
	result := db.Where("id = ?", id).First(&host)
	if err := db.Model(&host).Association("Tags").Clear(); err != nil {
		log.Println("clear association error for tags and host.")
	}
	result = db.Delete(&host)
	if result.Error != nil {
		log.Println(result.Error)
		return false
	}

	return true
}

func InsertHost(hostname string, user string, addr string, port int, password string, groupId int, tags []string, keyText string) *Host {
	var tagObjs []Tag
	for _, tagIdStr := range tags {
		tagId, _ := strconv.Atoi(tagIdStr)
		tag := Tag{}
		err := db.Where("id = ?", tagId).First(&tag)
		tagObjs = append(tagObjs, tag)
		if err != nil {
			log.Println(err)
		}
	}
	host := Host{
		Name:     hostname,
		User:     user,
		Addr:     addr,
		Port:     port,
		PassWord: password,
		KeyFile:  keyText,
		Tags:     tagObjs,
	}
	result := db.Omit("GroupId").Create(&host)
	if groupId != 0 {
		host.GroupId = groupId
		db.Save(&host)
	}
	if result.Error != nil {
		log.Println(result.Error)
	}
	return &host
}

func UpdateHost(id int, hostname string, user string, addr string, port int, password string, groupId int, tags []string, keyText string) *Host {
	host := Host{Id: id}
	result := db.Where("id = ?", id).First(&host)
	if result.Error != nil {
		log.Println(result.Error)
	}
	var tagObjs []Tag
	if err := db.Model(&host).Association("Tags").Clear(); err != nil {
		log.Println("clear association error for tags and host.")
	}
	for _, tagIdStr := range tags {
		tagId, _ := strconv.Atoi(tagIdStr)
		tag := Tag{}
		result := db.Where("id = ?", tagId).First(&tag)
		tagObjs = append(tagObjs, tag)
		if result.Error != nil {
			log.Println(result.Error)
		}
	}
	host.Tags = tagObjs
	if hostname != "" {
		host.Name = hostname
	}
	if user != "" {
		host.User = user
	}
	if port != 0 {
		host.Port = port
	}
	if addr != "" {
		host.Addr = addr
	}
	if password != "" {
		host.PassWord = password
	}
	if keyText != host.KeyFile {
		host.KeyFile = keyText
	}
	if groupId != 0 {
		host.GroupId = groupId
		result = db.Save(&host)
	} else {
		if err := db.Model(&host).Association("Group").Clear(); err != nil {
			log.Println(err)
		}
		result = db.Omit("GroupId").Save(&host)
	}
	if result.Error != nil {
		log.Println(result.Error)
	}
	return &host
}

func GetAllHost() []*Host {
	var hosts []*Host
	result := db.Preload("Tags").Preload("Group").Find(&hosts)
	if result.Error != nil {
		log.Println(result.Error)
	}

	return hosts
}

func GetHostByGlob(glob string) []*Host {
	var hosts []*Host
	glob = strings.Replace(glob, "*", "%", -1)
	result := db.Where("addr LIKE ?", glob).Find(&hosts)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return hosts
}

func GetHostByReg(regStr string) []*Host {
	var hosts []*Host
	var hostsR []*Host
	result := db.Find(&hosts)
	if result.Error != nil {
		log.Println(result.Error)
	}
	for i := 0; i < len(hosts); i++ {
		match, _ := regexp.MatchString(regStr, hosts[i].Addr)
		if match {
			hostsR = append(hostsR, hosts[i])
		}
	}

	return hostsR
}

func GetHostByAddr(addr string) []*Host {
	var hosts []*Host
	result := db.Where("addr = ?", addr).Find(&hosts)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return hosts
}

func GetHostByKeyFile(KeyFile string) int {
	var hosts []*Host
	result := db.Where("key_file = ?", KeyFile).Find(&hosts)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return len(hosts)
}

func GetHostsByTag(tag *Tag) []*Host {
	var hosts []*Host
	err := db.Model(&tag).Association("Hosts").Find(&hosts)
	if err != nil {
		log.Println(err)
	}
	return hosts
}

func GetHostsByGroup(group *Group) []*Host {
	var hosts []*Host
	result := db.Where("group_id = ?", group.Id).Preload("Tags").Preload("Group").Find(&hosts)
	if result.Error != nil {
		log.Println(result.Error)
	}
	return hosts
}

func UpdateHostStatus(host *Host) {
	_ = db.Omit("GroupId").Save(&host)
}
