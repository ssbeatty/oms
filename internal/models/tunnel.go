package models

type Tunnel struct {
	Id          int    `json:"id"`
	Mode        string `gorm:"size:64" json:"mode"`
	Source      string `gorm:"size:128;not null" json:"source"`
	Destination string `gorm:"size:128;not null" json:"destination"`
	Status      int    `gorm:"default:0" json:"status"`
	ErrorMsg    string `gorm:"size:512" json:"error_msg"`
	HostId      int    `json:"host_id"`
	Host        Host   `json:"-"`
}

func GetAllTunnel() ([]*Tunnel, error) {
	var tunnels []*Tunnel
	err := db.Preload("Host").Find(&tunnels).Error
	if err != nil {
		return nil, err
	}
	return tunnels, nil
}

func GetTunnelById(id int) (*Tunnel, error) {
	tunnel := Tunnel{}
	err := db.Where("id = ?", id).Preload("Host").First(&tunnel).Error
	if err != nil {
		return nil, err
	}
	return &tunnel, nil
}

func ExistedTunnel(id int) bool {
	var tunnel *Tunnel
	err := db.Where("id = ?", id).First(&tunnel).Error
	if err != nil {
		return false
	}
	if tunnel == nil {
		return false
	}
	return true
}

func InsertTunnel(mode, src, dest string, host *Host) (*Tunnel, error) {
	tunnel := Tunnel{
		Mode:        mode,
		Source:      src,
		Destination: dest,
		HostId:      host.Id,
	}
	err := db.Preload("Host").Create(&tunnel).Error
	if err != nil {
		return nil, err
	}
	return &tunnel, nil
}

func UpdateTunnel(id int, mode, src, dest string) (*Tunnel, error) {
	tunnel := Tunnel{Id: id}
	err := db.Where("id = ?", id).First(&tunnel).Error
	if err != nil {
		return nil, err
	}
	if mode != "" {
		tunnel.Mode = mode
	}
	if src != "" {
		tunnel.Source = src
	}
	if dest != "" {
		tunnel.Destination = dest
	}
	err = db.Save(&tunnel).Error
	if err != nil {
		return nil, err
	}
	return &tunnel, nil
}

func UpdateTunnelStatus(id int, status bool, msg string) (*Tunnel, error) {
	db.Lock()
	defer db.Unlock()

	tunnel := Tunnel{Id: id}
	err := db.Where("id = ?", id).First(&tunnel).Error
	if err != nil {
		return nil, err
	}
	if status {
		tunnel.Status = 1
	}

	if msg != "" {
		tunnel.ErrorMsg = msg
	}
	err = db.Save(&tunnel).Error
	if err != nil {
		return nil, err
	}
	return &tunnel, nil
}

func DeleteTunnelById(id int) error {
	tunnel := Tunnel{}
	err := db.Where("id = ?", id).First(&tunnel).Error
	if err != nil {
		return err
	}
	err = db.Delete(&tunnel).Error
	if err != nil {
		return err
	}
	return nil
}
