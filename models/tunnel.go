package models

type Tunnel struct {
	Id          int
	Mode        string `gorm:"size:64" json:"mode"`
	Source      string `gorm:"size:128;not null" json:"source"`
	Destination string `gorm:"size:128;not null" json:"destination"`
	Status      bool   `json:"status"`
	ErrorMsg    string `gorm:"size:128" json:"error_msg"`
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

func ExistedTunnel(name string) bool {
	var tunnels []*Tunnel
	err := db.Where("name = ?", name).Find(&tunnels).Error
	if err != nil {
		return false
	}
	if len(tunnels) == 0 {
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
	err = db.Preload("Host").Save(&tunnel).Error
	if err != nil {
		return nil, err
	}
	return &tunnel, nil
}

func UpdateTunnelStatus(id int, status bool, msg string) (*Tunnel, error) {
	tunnel := Tunnel{Id: id}
	err := db.Where("id = ?", id).First(&tunnel).Error
	if err != nil {
		return nil, err
	}
	tunnel.Status = status

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
