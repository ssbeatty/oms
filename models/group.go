package models

type Group struct {
	Id     int
	Name   string `gorm:"size:256;not null"`
	Mode   int    `gorm:"default:0;not null"` //0.主机模式, 1.其他匹配模式主机不生效
	Host   []Host
	Params string
}

func GetAllGroup() ([]*Group, error) {
	var groups []*Group
	err := db.Find(&groups).Error
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func GetGroupById(id int) (*Group, error) {
	group := Group{}
	err := db.Where("id = ?", id).First(&group).Error

	if err != nil {
		return nil, err
	}
	return &group, nil
}

func ExistedGroup(name string) bool {
	var groups []*Group
	err := db.Where("name = ?", name).Find(&groups).Error
	if err != nil {
		return false
	}
	if len(groups) == 0 {
		return false
	}
	return true
}

func InsertGroup(name string, params string, mode int) (*Group, error) {
	group := Group{
		Name:   name,
		Params: params,
		Mode:   mode,
	}
	err := db.Create(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func UpdateGroup(id int, name string, params string, mode int) (*Group, error) {
	group := Group{}
	err := db.Where("id = ?", id).FirstOrCreate(&group).Error
	if err != nil {
		return nil, err
	}
	if name != "" {
		group.Name = name
	}
	if params != "" {
		group.Params = params
	}
	if mode > 0 {
		group.Mode = mode
	}
	err = db.Save(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func DeleteGroupById(id int) error {
	err := db.Delete(&Group{}, id).Error
	if err != nil {
		return err
	}
	return nil
}
