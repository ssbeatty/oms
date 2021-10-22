package models

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"oms/conf"
)

var db *gorm.DB

func init() {
	var err error
	var dataSource string

	userName := conf.DefaultConf.MysqlConf.UserName
	passWord := conf.DefaultConf.MysqlConf.PassWord
	mysqlUrl := conf.DefaultConf.MysqlConf.Urls
	dbName := conf.DefaultConf.MysqlConf.DbName

	if conf.DefaultConf.DbDriver == "mysql" {
		dataSource = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", userName, passWord, mysqlUrl, dbName)
		db, err = gorm.Open(mysql.Open(dataSource), &gorm.Config{})
	} else {
		dataSource = "oms.db"
		db, err = gorm.Open(sqlite.Open(dataSource), &gorm.Config{})
	}

	if err != nil {
		log.Panicf("dataSource load error! exit! err: %v", err)
	}
	if err = db.AutoMigrate(new(Tag), new(Group), new(Host)); err != nil {
		log.Errorf("Migrate error! err: %v", err)
	}
}
