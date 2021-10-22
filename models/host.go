package models

import (
	log "github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
)

// Host Struct
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
	Group   Group `gorm:"constraint:OnDelete:SET NULL;"`
	Tags    []Tag `gorm:"many2many:host_tag"`
}

func GetHostById(id int) (*Host, error) {
	host := Host{}
	err := db.Where("id = ?", id).Preload("Tags").Preload("Group").First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func ExistedHost(name string, addr string) bool {
	var hosts []*Host
	err := db.Where(&Host{Name: name, Addr: addr}).Find(&hosts).Error
	if err != nil {
		return false
	}
	if len(hosts) == 0 {
		return false
	}
	return true
}

func DeleteHostById(id int) error {
	host := Host{}
	err := db.Where("id = ?", id).First(&host).Error
	if err != nil {
		return err
	}
	if err := db.Model(&host).Association("Tags").Clear(); err != nil {
		log.Errorf("DeleteHostById error Association tag Clear, err: %v", err)
	}
	err = db.Delete(&host).Error
	if err != nil {
		return err
	}

	return nil
}

func InsertHost(hostname string, user string, addr string, port int, password string, groupId int, tags []string, keyText string) (*Host, error) {
	var tagObjs []Tag
	for _, tagIdStr := range tags {
		tagId, err := strconv.Atoi(tagIdStr)
		if err != nil {
			log.Errorf("InsertHost error when Atoi tagArray idRaw, idRaw: %s, err: %v", tagIdStr, err)
			continue
		}
		tag := Tag{}
		err = db.Where("id = ?", tagId).First(&tag).Error
		if err != nil {
			log.Errorf("InsertHost error when First tag, err: %v", err)
			continue
		}
		tagObjs = append(tagObjs, tag)
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
	err := db.Omit("GroupId").Create(&host).Error
	if err != nil {
		return nil, err
	}
	if groupId != 0 {
		host.GroupId = groupId
		db.Save(&host)
	}
	return &host, nil
}

func UpdateHost(id int, hostname string, user string, addr string, port int, password string, groupId int, tags []string, keyText string) (*Host, error) {
	host := Host{Id: id}
	err := db.Where("id = ?", id).First(&host).Error
	if err != nil {
		return nil, err
	}

	if len(tags) > 0 {
		var tagObjs []Tag
		for _, tagIdStr := range tags {
			tagId, err := strconv.Atoi(tagIdStr)
			if err != nil {
				log.Errorf("UpdateHost error when Atoi tagIdStr, err: %v", err)
				continue
			}
			tag := Tag{}
			err = db.Where("id = ?", tagId).First(&tag).Error
			if err != nil {
				log.Errorf("UpdateHost error when First tag, err: %v", err)
				continue
			}
			tagObjs = append(tagObjs, tag)
		}
		if len(tagObjs) != 0 {
			if err := db.Model(&host).Association("Tags").Clear(); err != nil {
				log.Errorf("UpdateHost error when Association tag Clear, err: %v", err)
			}
			host.Tags = tagObjs
		}
	}

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
		if err := db.Save(&host).Error; err != nil {
			return nil, err
		}
	} else {
		if err := db.Model(&host).Association("Group").Clear(); err != nil {
			log.Errorf("UpdateHost error when Association group Clear, err: %v", err)
		}
		if err = db.Omit("GroupId").Save(&host).Error; err != nil {
			return nil, err
		}
	}
	return &host, nil
}

func GetAllHost() ([]*Host, error) {
	var hosts []*Host
	err := db.Preload("Tags").Preload("Group").Find(&hosts).Error
	if err != nil {
		return nil, err
	}

	return hosts, nil
}

func GetHostByGlob(glob string) ([]*Host, error) {
	var hosts []*Host
	glob = strings.Replace(glob, "*", "%", -1)
	err := db.Where("addr LIKE ?", glob).Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func GetHostByReg(regStr string) ([]*Host, error) {
	var hosts []*Host
	var hostsR []*Host
	err := db.Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(hosts); i++ {
		match, _ := regexp.MatchString(regStr, hosts[i].Addr)
		if match {
			hostsR = append(hostsR, hosts[i])
		}
	}

	return hostsR, nil
}

func GetHostByAddr(addr string) ([]*Host, error) {
	var hosts []*Host
	err := db.Where("addr = ?", addr).Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func GetHostsByTag(tag *Tag) ([]*Host, error) {
	var hosts []*Host
	err := db.Model(&tag).Association("Hosts").Find(&hosts)
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func GetHostsByGroup(group *Group) ([]*Host, error) {
	var hosts []*Host
	err := db.Where("group_id = ?", group.Id).Preload("Tags").Preload("Group").Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func UpdateHostStatus(host *Host) error {
	if err := db.Omit("GroupId").Save(&host).Error; err != nil {
		return err
	}
	return nil
}
