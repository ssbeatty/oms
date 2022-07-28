package models

import (
	"gorm.io/gorm/clause"
	"regexp"
	"strings"
)

type HostExport struct {
	Name        string   `csv:"name"`
	User        string   `csv:"user"`
	Addr        string   `csv:"addr"`
	Port        int      `csv:"port"`
	VNCPort     int      `csv:"vnc_port"`
	PassWord    string   `csv:"password"`
	Group       string   `csv:"group"`
	GroupParams string   `csv:"group_params"`
	Tags        []string `csv:"tags"`
	KeyFile     string   `csv:"key_file"`
	KeyName     string   `csv:"key_name"`
	KeyPhrase   string   `csv:"key_phrase"`
}

// Host Struct
type Host struct {
	Id           int        `json:"id"`
	Name         string     `gorm:"size:128;not null" json:"name"`
	User         string     `gorm:"size:128;not null" json:"user"`
	Addr         string     `gorm:"size:128;not null" json:"addr"`
	Port         int        `gorm:"default:22" json:"port"`
	VNCPort      int        `gorm:"default:5900" json:"vnc_port"`
	PassWord     string     `gorm:"size:128;not null" json:"-"`
	Status       bool       `gorm:"default:false" json:"status"`
	PrivateKey   PrivateKey `gorm:"constraint:OnDelete:SET NULL;" json:"-"`
	PrivateKeyID int        `json:"private_key_id"`
	GroupId      int        `json:"group_id"`
	Group        Group      `gorm:"constraint:OnDelete:SET NULL;" json:"group"`
	Tags         []Tag      `gorm:"many2many:host_tag" json:"tags"`
	Tunnels      []Tunnel   `gorm:"constraint:OnDelete:CASCADE;" json:"tunnels"`
}

func ParseHostList(pType string, id int) ([]*Host, error) {
	var hosts []*Host
	if pType == "host" {
		host, err := GetHostById(id)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	} else if pType == "tag" {
		tag, err := GetTagById(id)
		if err != nil {
			return nil, err
		}
		hosts, err = GetHostsByTag(tag)
		if err != nil {
			return nil, err
		}
	} else {
		group, err := GetGroupById(id)
		if err != nil {
			return nil, err
		}
		if group.Mode == 0 {
			hosts, err = GetHostsByGroup(group)
			if err != nil {
				return nil, err
			}
		} else {
			args := strings.Split(group.Params, " ")
			if len(args) < 2 {
				return nil, err
			} else {
				if strings.Contains(args[1], "\"") {
					args[1] = strings.ReplaceAll(args[1], "\"", "")
				}
			}
			switch args[0] {
			case "-G":
				hosts, err := GetHostByGlob(args[1])
				if err != nil {
					return nil, err
				}
				return hosts, nil
			case "-L":
				var hosts []*Host
				addrArgs := strings.Split(args[1], ",")
				for _, addr := range addrArgs {
					host, err := GetHostByAddr(addr)
					if err != nil {
						return nil, err
					}
					hosts = append(hosts, host...)
				}
				return hosts, nil
			case "-E":
				hosts, err := GetHostByReg(args[1])
				if err != nil {
					return nil, err
				}
				return hosts, nil
			default:
				hosts, err := GetHostByGlob(args[0])
				if err != nil {
					return nil, err
				}
				return hosts, nil
			}
		}

	}
	return hosts, nil
}

func GetHostByIdWithPreload(id int) (*Host, error) {
	host := Host{}
	err := db.Where("id = ?", id).Preload(clause.Associations).First(&host).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func GetHostById(id int) (*Host, error) {
	host := Host{}
	err := db.Where("id = ?", id).First(&host).Error
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

func DeleteHostById(id int) (*Host, error) {
	host := Host{}
	err := db.Where("id = ?", id).First(&host).Error
	if err != nil {
		return nil, err
	}
	if err := db.Model(&host).Association("Tags").Clear(); err != nil {
		log.Errorf("DeleteHostById error Association tag Clear, err: %v", err)
	}
	err = db.Delete(&host).Error
	if err != nil {
		return nil, err
	}

	return &host, nil
}

func InsertHost(hostname string, user string, addr string, port int, password string, groupId int, tags []int, privateKeyID, vncPort int) (*Host, error) {
	var tagObjs []Tag
	for _, tagId := range tags {
		tag := Tag{}
		err := db.Where("id = ?", tagId).First(&tag).Error
		if err != nil {
			log.Errorf("InsertHost error when First tag, err: %v", err)
			continue
		}
		tagObjs = append(tagObjs, tag)
	}
	host := Host{
		Name:         hostname,
		User:         user,
		Addr:         addr,
		Port:         port,
		PassWord:     password,
		PrivateKeyID: privateKeyID,
		Tags:         tagObjs,
		VNCPort:      vncPort,
	}
	err := db.Omit("GroupId", "PrivateKeyID").Create(&host).Error
	if err != nil {
		return nil, err
	}
	if groupId != 0 {
		group := Group{}
		err := db.Where("id = ?", groupId).First(&group).Error
		if err != nil {
			return &host, nil
		}
		host.GroupId = groupId
		db.Omit("PrivateKeyID").Save(&host)
		host.Group = group
	}
	if privateKeyID != 0 {
		privateKey := PrivateKey{}
		err := db.Where("id = ?", privateKeyID).First(&privateKey).Error
		if err != nil {
			return &host, nil
		}
		host.PrivateKeyID = privateKeyID
		db.Omit("GroupId").Save(&host)
		host.PrivateKey = privateKey
	}
	return &host, nil
}

func UpdateHost(id int, hostname string, user string, addr string, port int, password string, groupId int, tags []int, privateKeyID, vncPort int) (*Host, error) {
	host := Host{Id: id}
	err := db.Where("id = ?", id).First(&host).Error
	if err != nil {
		return nil, err
	}

	if len(tags) > 0 {
		var tagObjs []Tag
		for _, tagId := range tags {
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
	} else {
		if err := db.Model(&host).Association("Tags").Clear(); err != nil {
			log.Errorf("UpdateHost error when Association tag Clear, err: %v", err)
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
	if vncPort != 0 {
		host.VNCPort = vncPort
	}
	if groupId != 0 {
		group := Group{}
		err := db.Where("id = ?", groupId).First(&group).Error
		if err == nil {
			host.Group = group
		}
		if err := db.Omit("PrivateKeyID").Save(&host).Error; err != nil {
			return nil, err
		}
	} else {
		if err := db.Model(&host).Association("Group").Clear(); err != nil {
			log.Errorf("UpdateHost error when Association group Clear, err: %v", err)
		}
		if err = db.Omit("GroupId", "PrivateKeyID").Save(&host).Error; err != nil {
			return nil, err
		}
	}
	if privateKeyID != 0 {
		privateKey := PrivateKey{}
		err := db.Where("id = ?", privateKeyID).First(&privateKey).Error
		if err == nil {
			host.PrivateKey = privateKey
		}
		if err := db.Omit("GroupId").Save(&host).Error; err != nil {
			return nil, err
		}
	} else {
		if err := db.Model(&host).Association("PrivateKey").Clear(); err != nil {
			log.Errorf("UpdateHost error when Association privateKey Clear, err: %v", err)
		}
		if err = db.Omit("GroupId", "PrivateKeyID").Save(&host).Error; err != nil {
			return nil, err
		}
	}
	return &host, nil
}

func GetAllHost() ([]*Host, error) {
	var hosts []*Host
	err := db.Preload(clause.Associations).Find(&hosts).Error
	if err != nil {
		return nil, err
	}

	return hosts, nil
}

func GetAllHostWithOutPreload() ([]*Host, error) {
	var hosts []*Host
	err := db.Find(&hosts).Error
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
	err := db.Where("group_id = ?", group.Id).Find(&hosts).Error
	if err != nil {
		return nil, err
	}
	return hosts, nil
}

func UpdateHostStatus(host *Host) error {
	db.Lock()
	defer db.Unlock()

	if err := db.Select("Status").Save(&host).Error; err != nil {
		return err
	}
	return nil
}
