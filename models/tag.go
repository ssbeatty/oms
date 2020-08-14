package models

type Tag struct {
	Id    int
	Name  string  `orm:"size(100)"`
	Hosts []*Host `orm:"reverse(many);null"`
}
