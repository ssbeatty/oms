package models

import (
	_ "github.com/go-sql-driver/mysql" // import your used driver
)

// Model Struct
type Host struct {
	Id       int
	Name     string `orm:"size(100)"`
	Addr     string `orm:"null"`
	Port     int    `orm:"null"`
	PassWord string `orm:"null"`
	KeyFile  string `orm:"null"`
}
