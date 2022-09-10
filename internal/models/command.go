package models

import (
	"errors"
	"gorm.io/gorm"
	"strings"
)

type CommandHistory struct {
	Id    int    `json:"id"`
	Cmd   string `gorm:"not null;index:idx_cmd;unique" json:"cmd"`
	Times uint32 `gorm:"index:idx_times" json:"times"`
}

func InsertOrUpdateCommandHistory(cmd string) error {
	tx := db.Begin()
	defer tx.Commit()

	ch := CommandHistory{
		Cmd:   cmd,
		Times: 1,
	}
	result := tx.Where("cmd = ?", cmd).First(&ch)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			err := tx.Create(&ch).Error
			if err != nil {
				return err
			}
			return nil
		} else {
			return result.Error
		}
	}
	ch.Times++

	if err := tx.Save(&ch).Error; err != nil {
		return err
	}

	return nil
}

func DeleteCommandHistoryById(id int) error {
	ch := CommandHistory{
		Id: id,
	}
	err := db.Delete(&ch).Error
	if err != nil {
		return err
	}
	return nil
}

func SearchCommandHistory(keyword string, limit int) ([]*CommandHistory, error) {
	var (
		err     error
		records []*CommandHistory
	)
	if limit == 0 {
		limit = 10
	}

	keyword = strings.TrimRight(keyword, " ")

	if keyword == "" {
		err = db.Order("times DESC").Limit(limit).Find(&records).Error
	} else {
		arg := keyword + "%"
		err = db.Where("cmd LIKE ?", arg).Order("times DESC").Limit(limit).Find(&records).Error
	}

	if err != nil {
		return nil, err
	}
	return records, nil
}
