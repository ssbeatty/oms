package models

type Group struct {
	Id     int
	Name   string  `orm:"size(100)"`
	Mode   int     `orm:"default(0)"` //0.Default mode & use Host, 1.Use other func
	Host   []*Host `orm:"reverse(many);null"`
	Params string  `orm:"null" json:"-"`
}
