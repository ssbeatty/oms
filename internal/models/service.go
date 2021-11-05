package models

import (
	"fmt"
	"github.com/ssbeatty/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"oms/pkg/logger"
	"sync"
)

var db *DataBase
var log *logger.Logger

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

func InitModels(dsn, dbName, user, pass, driver string) {
	var d *gorm.DB
	var err error
	var dataSource string

	log = logger.NewLogger("db")

	if driver == "mysql" {
		dataSource = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", user, pass, dsn, dbName)
		d, err = gorm.Open(mysql.Open(dataSource), &gorm.Config{})
		db = &DataBase{d, nil}
	} else {
		dataSource = "oms.db"
		d, err = gorm.Open(sqlite.Open(dataSource), &gorm.Config{})
		// 防止database locked
		db = &DataBase{d, &sync.Mutex{}}
	}

	if err != nil {
		log.Errorf("dataSource load error! exit! err: %v", err)
	}
	if err = db.AutoMigrate(new(Tag), new(Group), new(Host), new(Tunnel), new(Job)); err != nil {
		log.Errorf("Migrate error! err: %v", err)
	}
}
