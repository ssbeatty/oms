package models

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"oms/logger"
)

type User struct {
	Id int64
	NickName string `orm:"null"`
	Name string `orm:"size(100)"`
	Phone string `orm:"size(20);unique"`
	Ctime string `orm:"size(100)"`
	Emil string `orm:"size(100);null"`
	IsActive int `orm:"default(1)"`
	UserPassword string `json:"-"`
}

func GetUserId(id int64) *User {
	var o = orm.NewOrm()
	user := User{Id: id}
	err := o.Read(&user)
	if err == orm.ErrNoRows {
		logger.Logger.Println(err)
	} else if err == orm.ErrMissPK {
		logger.Logger.Println(err)
	} else {
		logger.Logger.Println(user)
	}
	if err != nil {
		logger.Logger.Println(err)
	}
	return &user
}

func GetUserPhone(phone string) *User {
	var o = orm.NewOrm()
	user := User{Phone: phone}
	fmt.Println(user)
	err := o.Read(&user, "phone")
	if err == orm.ErrNoRows {
		logger.Logger.Println(err)
	} else if err == orm.ErrMissPK {
		logger.Logger.Println(err)
	} else {
		logger.Logger.Println(user)
	}
	if err != nil {
		logger.Logger.Println(err)
	}
	return &user
}


func DeleteUser(userid int64) error {
	var o = orm.NewOrm()
	user := User{Id: userid}
	user.IsActive = 0
	if num, err := o.Update(&user); err == nil {
		logger.Logger.Fatalln("update %s succeed, %d line", user,num)
	}
	return fmt.Errorf("delete user faild")
}

func UpdateUser(user map[string]interface{}) error {
	var o = orm.NewOrm()
	_, err := o.QueryTable("user").Update(user)
	return err
}

func InsertUser(user *User) error {
	var o = orm.NewOrm()
	if num, err := o.Insert(&user); err == nil {
		logger.Logger.Fatalln("update %s succeed, %d line", user,num)

	}
	return fmt.Errorf("delete user faild")
}


