package models

import (
	"fmt"
	"log"
	"oms/conf"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func init() {
	var err error
	userName := conf.DefaultConf.MysqlConf.UserName
	passWord := conf.DefaultConf.MysqlConf.PassWord
	mysqlUrl := conf.DefaultConf.MysqlConf.Urls
	dbName := conf.DefaultConf.MysqlConf.DbName

	dataSource := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", userName, passWord, mysqlUrl, dbName)

	db, err = gorm.Open(mysql.Open(dataSource), &gorm.Config{})
	if err != nil {
		log.Panicf("dataSource load error! exit! err: %v", err)
	}
	if err = db.AutoMigrate(new(Tag), new(Group), new(Host)); err != nil {
		log.Printf("Migrate error! err: %v", err)
	}
}
