package conf

import (
	"github.com/gobuffalo/packr"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type Conf struct {
	MysqlConf MysqlConfig `yaml:"mysql"`
	AppConf   AppConfig   `yaml:"app"`
}

type MysqlConfig struct {
	UserName string `yaml:"user"`
	PassWord string `yaml:"password"`
	Urls     string `yaml:"urls"`
	DbName   string `yaml:"db"`
}

type AppConfig struct {
	AppName  string `yaml:"name"`
	HttpAddr string `yaml:"addr"`
	HttpPort int    `yaml:"port"`
	RunMode  string `yaml:"mode"`
}

var DefaultConf *Conf

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func init() {
	var yamlFile []byte
	var err error
	DefaultConf = new(Conf)
	if ok, _ := PathExists("config.yaml"); ok == true {
		yamlFile, err = ioutil.ReadFile("config.yaml")
		if err != nil {
			log.Printf("yamlFile.Get err #%v ", err)
		}
	} else {
		box := packr.NewBox("./")
		yamlFile, err = box.Find("config.yaml.example")
	}

	err = yaml.Unmarshal(yamlFile, DefaultConf)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}
