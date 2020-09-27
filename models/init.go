package models

import (
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"

	_ "github.com/go-sql-driver/mysql"
)

func init() {

	userName := beego.AppConfig.String("mysqluser")
	passWord := beego.AppConfig.String("mysqlpass")
	mysqlUrl := beego.AppConfig.String("mysqlurls")
	dbName := beego.AppConfig.String("mysqldb")

	dataSource := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8", userName, passWord, mysqlUrl, dbName)

	// set default database
	orm.RegisterDataBase("default", "mysql", dataSource, 30)

	// register model
	orm.RegisterModel(new(Host), new(Group), new(Tag), new(User))

	// print sql lin
	orm.Debug = true

	// create table
	orm.RunSyncdb("default", false, true)
}
