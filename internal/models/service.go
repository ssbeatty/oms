package models

import (
	"errors"
	"fmt"
	"github.com/ssbeatty/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"oms/internal/utils"
	"oms/pkg/logger"
	"os"
	"path"
	"strings"
	"sync"
)

const (
	defaultSort = "id ASC"
)

var (
	db       *DataBase
	dataPath string
	log      *logger.Logger
)

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

func InitModels(dsn, dbName, user, pass, driver, _dataPath string) error {
	var d *gorm.DB
	var err error
	var dataSource string

	dataPath = _dataPath
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
		if exist, _ := utils.PathExists(dataPath); !exist {
			_ = os.MkdirAll(dataPath, os.ModePerm)
		}

		dataSource = path.Join(dataPath, "oms.db")
		d, err = gorm.Open(sqlite.Open(dataSource), &gorm.Config{})
		// 防止database locked
		db = &DataBase{d, &sync.Mutex{}}
	}

	if err != nil {
		log.Errorf("dataSource load error! exit! err: %v", err)
		return err
	}
	if err = db.AutoMigrate(
		new(Tag), new(Group), new(Host), new(Tunnel), new(Job), new(PrivateKey), new(TaskInstance), new(PlayBook),
	); err != nil {
		log.Errorf("Migrate error! err: %v", err)
		return err
	}

	return nil
}

func GetPaginateQuery[T *TaskInstance | *[]*TaskInstance | *Host | *[]*Host](
	instance T, pageSize, page int, params map[string]interface{}, preload bool) (int64, error) {
	var total int64

	switch {
	case pageSize <= 0:
		pageSize = 20
	case page <= 0:
		page = 1
	}

	offset := (page - 1) * pageSize

	d := db.Where(params)
	d.Model(instance).Count(&total)

	if preload {
		d = d.Preload(clause.Associations)
	}

	d = d.Order(defaultSort).Offset(offset).Limit(pageSize).Find(instance)

	return total, d.Error
}
