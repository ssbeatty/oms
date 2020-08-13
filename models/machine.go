package models

import (
	_ "github.com/go-sql-driver/mysql" // import your used driver
)

// Model Struct
type Host struct {
	Id       int
	Name     string `orm:"size(100)"`
	Addr     string `orm:"null"`
	Port     int    `orm:"default(22)"`
	PassWord string `orm:"null" json:"-"`
	KeyFile  string `orm:"null" json:"-"`

	Group *Group `orm:"rel(fk);null"`
	Tags  []*Tag `orm:"rel(m2m);null"`
}

type Group struct {
	Id   int
	Name string  `orm:"size(100)"`
	Mode int     `orm:"default(0)"` //0.Default mode & use Host, 1.Use other func
	Host []*Host `orm:"reverse(many);null"`
}

type Tag struct {
	Id    int
	Name  string  `orm:"size(100)"`
	Hosts []*Host `orm:"reverse(many);null"`
}
