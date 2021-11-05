package models

type Job struct {
	Id     int    `json:"id"`
	Name   string `gorm:"size:128" json:"name"`
	Type   string `gorm:"size:32;not null" json:"type"`
	Spec   string `gorm:"size:128" json:"spec"`
	Cmd    string `gorm:"size:512" json:"cmd"`
	Status string `gorm:"size:64;default: ready" json:"status"`
	HostId int    `json:"host_id"`
	Host   Host   `json:"-"`
}

func GetJobByHostId(id int) (*Job, error) {
	job := Job{}
	err := db.Where("host_id = ?", id).Preload("Host").First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func GetAllJob() ([]*Job, error) {
	var jobs []*Job
	err := db.Preload("Host").Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func GetJobById(id int) (*Job, error) {
	job := Job{}
	err := db.Where("id = ?", id).Preload("Host").First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func ExistedJob(id int) bool {
	var job *Job
	err := db.Where("id = ?", id).First(&job).Error
	if err != nil {
		return false
	}
	if job == nil {
		return false
	}
	return true
}

func InsertJob(name, t, spec, cmd string, host *Host) (*Job, error) {
	job := Job{
		Name:   name,
		Type:   t,
		Spec:   spec,
		Cmd:    cmd,
		HostId: host.Id,
	}
	err := db.Preload("Host").Create(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func UpdateJob(id int, name, t, spec, cmd string) (*Job, error) {
	job := Job{Id: id}
	err := db.Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	if name != "" {
		job.Name = name
	}
	if t != "" {
		job.Type = t
	}
	if spec != "" {
		job.Spec = spec
	}
	if spec != "" {
		job.Cmd = cmd
	}
	err = db.Save(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func UpdateJobStatus(id int, status string) (*Job, error) {
	db.Lock()
	defer db.Unlock()

	job := Job{Id: id}
	err := db.Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	if status != "" {
		job.Status = status
	}

	err = db.Save(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func DeleteJobById(id int) error {
	job := Job{}
	err := db.Where("id = ?", id).First(&job).Error
	if err != nil {
		return err
	}
	err = db.Delete(&job).Error
	if err != nil {
		return err
	}
	return nil
}
