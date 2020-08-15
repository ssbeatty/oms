package models

import (
	"fmt"
	"github.com/astaxie/beego/orm"
)

func TestFunc() {
	var o = orm.NewOrm()
	//group := Group{Id: 1}
	//err := o.Read(&group)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(group)

	//group := Group{Id: 2}
	//_, err := o.Delete(&group)
	//if err != nil {
	//	fmt.Println(err)
	//}

	obj := Tag{Id: 3}
	m2m := o.QueryM2M(&obj, "Hosts")
	_, err := m2m.Clear()
	if err != nil {
		fmt.Println(err)
	}
	_, err = o.Delete(&obj)
	if err != nil {
		fmt.Println(err)
	}
}
