package config

import (
	"github.com/gobuffalo/packr"
	"gopkg.in/yaml.v2"
	"io/fs"
	"io/ioutil"
	"oms/internal/utils"
	"time"
)

const (
	defaultDataPath    = "data"
	DefaultTmpPath     = "tmp"
	DefaultTaskTmpPath = "tasks"
	UploadPath         = "upload"
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
	Name     string        `yaml:"name"`
	Addr     string        `yaml:"addr"`
	Port     int           `yaml:"port"`
	Mode     string        `yaml:"mode"`
	RunStart bool          `yaml:"run_start"`
	DataPath string        `yaml:"data_path"` // db file and tmp path
	TempDate time.Duration `yaml:"temp_date"`
}

// NewServerConfig 加载优先级路径 > 当前目录的config.yaml > 打包在可执行文件里的config.yaml.example
func NewServerConfig(path string) (*Conf, error) {
	var data []byte
	var err error
	ret := new(Conf)
	if path == "" {
		path = "config.yaml"
	}

	// 从本地读取 读不到从二进制静态文件包中读取
	if ok, _ := utils.PathExists(path); ok {
		data, err = ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
	} else {
		box := packr.NewBox("../../configs")
		data, err = box.Find("config.yaml.example")
		if err != nil {
			return nil, err
		}

		// config 写入当前目录
		_ = ioutil.WriteFile(path, data, fs.FileMode(0644))
	}

	err = yaml.Unmarshal(data, ret)
	if err != nil {
		return nil, err
	}

	if ret.App.DataPath == "" {
		ret.App.DataPath = defaultDataPath
	}

	return ret, nil
}
