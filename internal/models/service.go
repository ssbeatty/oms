package models

import (
	"errors"
	"fmt"
	"github.com/ssbeatty/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"oms/pkg/logger"
	"strings"
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

func InitModels(dsn, dbName, user, pass, driver string) error {
	var d *gorm.DB
	var err error
	var dataSource string

	log = logger.NewLogger("db")

	if driver == "mysql" {
		dataSource = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", user, pass, dsn, dbName)
		d, err = gorm.Open(mysql.Open(dataSource), &gorm.Config{})
		db = &DataBase{d, nil}
	} else if driver == "postgres" {
		dsnArgs := strings.Split(dsn, ":")
		if len(dsnArgs) < 2 {
			return errors.New("dsn parse error")
		}
		dataSource = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			dsnArgs[0], user, pass, dbName, dsnArgs[1],
		)
		d, err = gorm.Open(postgres.Open(dataSource), &gorm.Config{})
		db = &DataBase{d, nil}
	} else {
		dataSource = "oms.db"
		d, err = gorm.Open(sqlite.Open(dataSource), &gorm.Config{})
		// 防止database locked
		db = &DataBase{d, &sync.Mutex{}}
	}

	if err != nil {
		log.Errorf("dataSource load error! exit! err: %v", err)
		return err
	}
	if err = db.AutoMigrate(new(Tag), new(Group), new(Host), new(Tunnel), new(Job), new(PrivateKey)); err != nil {
		log.Errorf("Migrate error! err: %v", err)
		return err
	}

	return nil
}
