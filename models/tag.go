package models

type Tag struct {
	Id    int
	Name  string `gorm:"size:100;not null;unique"`
	Hosts []Host `gorm:"many2many:host_tag;"`
}

func GetAllTag() ([]*Tag, error) {
	var tags []*Tag
	err := db.Find(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func GetTagById(id int) (*Tag, error) {
	tag := Tag{}
	err := db.Where("id = ?", id).First(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func ExistedTag(name string) bool {
	var tags []*Tag
	err := db.Where("name = ?", name).Find(&tags).Error
	if err != nil {
		return false
	}
	if len(tags) == 0 {
		return false
	}
	return true
}

func InsertTag(name string) (*Tag, error) {
	tag := Tag{
		Name: name,
	}
	err := db.Create(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func UpdateTag(id int, name string) (*Tag, error) {
	tag := Tag{Id: id}
	err := db.Where("id = ?", id).First(&tag).Error
	if err != nil {
		return nil, err
	}
	if name != "" {
		tag.Name = name
	}
	err = db.Save(&tag).Error
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func DeleteTagById(id int) error {
	tag := Tag{}
	err := db.Where("id = ?", id).First(&tag).Error
	if err != nil {
		return err
	}
	if err := db.Model(&tag).Association("Hosts").Clear(); err != nil {
		return err
	}
	err = db.Delete(&tag).Error
	if err != nil {
		return err
	}
	return nil
}
