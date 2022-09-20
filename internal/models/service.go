package models

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4"
	"github.com/ssbeatty/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"oms/pkg/logger"
	"oms/pkg/utils"
	"os"
	"path"
	"strings"
	"sync"
)

const (
	defaultSort      = "id DESC"
	DBDriverMysql    = "mysql"
	DBDriverPostgres = "postgres"
	DBDriverSqlite   = "sqlite"
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

func createDB(driver, user, pass, dsn, dbName string) error {

	switch driver {
	case DBDriverSqlite:
		return nil
	case DBDriverMysql:
		dataSource := fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8", user, pass, dsn)
		d, err := sql.Open(driver, dataSource)
		if err != nil {
			return err
		}
		defer d.Close()

		_, err = d.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s CHARACTER SET utf8 COLLATE utf8_general_ci;", dbName))
		if err != nil {
			return err
		}
	case DBDriverPostgres:
		dsnArgs := strings.Split(dsn, ":")
		if len(dsnArgs) < 2 {
			return errors.New("dsn parse error")
		}
		dataSource := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
			dsnArgs[0], user, pass, dsnArgs[1],
		)
		d, err := sql.Open("pgx", dataSource)
		if err != nil {
			return err
		}
		defer d.Close()

		_, err = d.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			return err
		}
	}

	return nil
}

func InitModels(dsn, dbName, user, pass, driver, _dataPath string) error {
	var d *gorm.DB
	var err error
	var dataSource string

	dataPath = _dataPath
	log = logger.NewLogger("db")

	_ = createDB(driver, user, pass, dsn, dbName)
	dfConfig := &gorm.Config{}

	if driver == DBDriverMysql {
		dataSource = fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", user, pass, dsn, dbName)
		d, err = gorm.Open(mysql.Open(dataSource), dfConfig)
		db = &DataBase{d, nil}
	} else if driver == DBDriverPostgres {
		dsnArgs := strings.Split(dsn, ":")
		if len(dsnArgs) < 2 {
			return errors.New("dsn parse error")
		}
		dataSource = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			dsnArgs[0], user, pass, dbName, dsnArgs[1],
		)
		d, err = gorm.Open(postgres.Open(dataSource), dfConfig)
		db = &DataBase{d, nil}
	} else {
		if exist, _ := utils.PathExists(dataPath); !exist {
			_ = os.MkdirAll(dataPath, os.ModePerm)
		}

		dataSource = path.Join(dataPath, "oms.db")
		d, err = gorm.Open(sqlite.Open(dataSource), dfConfig)
		// 防止database locked
		db = &DataBase{d, &sync.Mutex{}}
	}

	if err != nil {
		log.Errorf("dataSource load error! exit! err: %v", err)
		return err
	}

	if err = db.AutoMigrate(
		new(Tag), new(Group), new(Host), new(Tunnel), new(Job), new(PrivateKey), new(TaskInstance), new(PlayBook),
		new(CommandHistory), new(QuicklyCommand),
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
