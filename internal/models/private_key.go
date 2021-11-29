package models

type PrivateKey struct {
	Id         int    `json:"id"`
	Name       string `gorm:"size:128;not null" json:"name"`
	KeyFile    string `gorm:"type:text" json:"key_file"`
	Passphrase string `json:"-"`
}

func GetAllPrivateKey() ([]*PrivateKey, error) {
	var privateKeys []*PrivateKey
	err := db.Find(&privateKeys).Error
	if err != nil {
		return nil, err
	}
	return privateKeys, nil
}

func GetPrivateKeyById(id int) (*PrivateKey, error) {
	privateKey := PrivateKey{}
	err := db.Where("id = ?", id).First(&privateKey).Error
	if err != nil {
		return nil, err
	}
	return &privateKey, nil
}

func InsertPrivateKey(name, keyFile, passphrase string) (*PrivateKey, error) {
	privateKey := PrivateKey{
		Name:       name,
		KeyFile:    keyFile,
		Passphrase: passphrase,
	}
	err := db.Create(&privateKey).Error
	if err != nil {
		return nil, err
	}
	return &privateKey, nil
}

func UpdatePrivateKey(id int, name, keyFile, passphrase string) (*PrivateKey, error) {
	privateKey := PrivateKey{Id: id}
	err := db.Where("id = ?", id).First(&privateKey).Error
	if err != nil {
		return nil, err
	}
	if name != "" {
		privateKey.Name = name
	}
	if keyFile != "" {
		privateKey.KeyFile = keyFile
	}
	if passphrase != "" {
		privateKey.Passphrase = passphrase
	}
	err = db.Save(&privateKey).Error
	if err != nil {
		return nil, err
	}
	return &privateKey, nil
}

func DeletePrivateKeyById(id int) error {
	privateKey := PrivateKey{}
	err := db.Where("id = ?", id).First(&privateKey).Error
	if err != nil {
		return err
	}
	err = db.Delete(&privateKey).Error
	if err != nil {
		return err
	}
	return nil
}
