package config

import (
	"github.com/gobuffalo/packr"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"oms/internal/utils"
)

type Conf struct {
	Db  DB  `yaml:"db"`
	App App `yaml:"app"`
}

type DB struct {
	Driver   string `yaml:"driver"`
	UserName string `yaml:"user"`
	PassWord string `yaml:"password"`
	Dsn      string `yaml:"dsn"`
	DbName   string `yaml:"db_name"`
}

type App struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// NewServerConfig 加载优先级路径 > 当前目录的config.yaml > 打包在可执行文件里的config.yaml.example
func NewServerConfig(path string) (*Conf, error) {
	var data []byte
	var err error
	ret := new(Conf)
	if path == "" {
		path = "config.yaml"
	}

	if ok, _ := utils.PathExists(path); ok {
		data, err = ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
	} else {
		box := packr.NewBox("../configs")
		data, err = box.Find("config.yaml.example")
		if err != nil {
			return nil, err
		}
	}

	err = yaml.Unmarshal(data, ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
