package models

import (
	"fmt"
	"path"
	"time"
)

type Job struct {
	Id          int            `json:"id"`
	Name        string         `gorm:"size:128" json:"name"`
	Type        string         `gorm:"size:32;not null" json:"type"`
	Spec        string         `gorm:"size:128" json:"spec"`
	Cmd         string         `gorm:"size:512" json:"cmd"`
	Status      string         `gorm:"size:64;default: ready" json:"status"`
	ExecuteID   int            `json:"execute_id"`
	ExecuteType string         `gorm:"size:64" json:"execute_type"`
	Instances   []TaskInstance `gorm:"constraint:OnDelete:CASCADE;" json:"instances"`
}

type TaskInstance struct {
	Id        int       `json:"id"`
	JobId     int       `json:"job_id"`
	Job       Job       `json:"-"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `gorm:"size:64;default: ready" json:"status"`
	LogPath   string    `gorm:"size:256" json:"log_path"`
	LogData   string    `gorm:"type:text" json:"log_data"`
}

func GetAllJob() ([]*Job, error) {
	var jobs []*Job
	err := db.Find(&jobs).Error
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func GetJobById(id int) (*Job, error) {
	job := Job{}
	err := db.Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func InsertJob(name, t, spec, cmd string, executeID int, executeType string) (*Job, error) {
	job := Job{
		Name:        name,
		Type:        t,
		Spec:        spec,
		Cmd:         cmd,
		ExecuteID:   executeID,
		ExecuteType: executeType,
	}
	err := db.Create(&job).Error
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
	if cmd != "" {
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

// RefreshJob 刷新job的状态
func RefreshJob(job *Job) error {
	err := db.First(&job).Error
	if err != nil {
		return err
	}

	return nil
}

func (ti *TaskInstance) GenerateLogPath(tmpPath string) string {
	return path.Join(tmpPath, fmt.Sprintf("%d.log", ti.Id))
}

func (ti *TaskInstance) UpdateStatus(status string) error {
	return db.Model(&TaskInstance{}).Where("id", ti.Id).Update("status", status).Error
}

func GetTaskInstanceByJob(jobId int) ([]*TaskInstance, error) {
	var instances []*TaskInstance
	err := db.Where("job_id", jobId).Find(&instances).Error
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func GetTaskInstanceById(id int) (*TaskInstance, error) {
	var instance *TaskInstance
	err := db.Preload("Job").Where("id", id).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func GetAllTaskInstance() ([]*TaskInstance, error) {
	var instances []*TaskInstance
	err := db.Find(&instances).Error
	if err != nil {
		return nil, err
	}
	return instances, nil
}

func UpdateTaskInstanceLogTrace(instance *TaskInstance, logPath string) error {
	instance.LogPath = logPath
	return db.Model(&TaskInstance{}).Where("id", instance.Id).Update("log_path", logPath).Error
}

func InsertTaskInstance(jobId int, start, end time.Time, logPath string) (*TaskInstance, error) {
	job, err := GetJobById(jobId)
	if err != nil {
		return nil, err
	}
	instance := TaskInstance{
		JobId:     jobId,
		StartTime: start,
		EndTime:   end,
		LogPath:   logPath,
	}
	err = db.Create(&instance).Error
	if err != nil {
		return nil, err
	}

	instance.Job = *job

	return &instance, nil
}
