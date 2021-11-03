package models

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/ssbeatty/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"oms/conf"
	"sync"
)

var db *DataBase

type DataBase struct {
	*gorm.DB
	mu *sync.Mutex
}

func (d *DataBase) Lock() {
	if d.mu != nil {
		d.mu.Lock()
	}
}

func (d *DataBase) Unlock() {
	if d.mu != nil {
		d.mu.Unlock()
	}
}

func init() {
	var d *gorm.DB
	var err error
	var dataSource string

	userName := conf.DefaultConf.MysqlConf.UserName
	passWord := conf.DefaultConf.MysqlConf.PassWord
	mysqlUrl := conf.DefaultConf.MysqlConf.Urls
	dbName := conf.DefaultConf.MysqlConf.DbName

	if conf.DefaultConf.DbDriver == "mysql" {
		dataSource = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", userName, passWord, mysqlUrl, dbName)
		d, err = gorm.Open(mysql.Open(dataSource), &gorm.Config{})
		db = &DataBase{d, nil}
	} else {
		dataSource = "oms.db"
		d, err = gorm.Open(sqlite.Open(dataSource), &gorm.Config{})
		// 防止database locked
		db = &DataBase{d, &sync.Mutex{}}
	}

	if err != nil {
		log.Panicf("dataSource load error! exit! err: %v", err)
	}
	if err = db.AutoMigrate(new(Tag), new(Group), new(Host), new(Tunnel), new(Job)); err != nil {
		log.Errorf("Migrate error! err: %v", err)
	}
}
