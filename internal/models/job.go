package models

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io/ioutil"
	"oms/internal/config"
	"os"
	"path"
	"path/filepath"
	"time"
)

const (
	InstanceStatusRunning = "running"
	InstanceStatusDone    = "done"
)

type Job struct {
	Id          int            `json:"id"`
	Name        string         `gorm:"size:128" json:"name"`
	Type        string         `gorm:"size:32;not null" json:"type"`
	Spec        string         `gorm:"size:128" json:"spec"`
	Cmd         string         `gorm:"size:512" json:"cmd"`
	CmdId       int            `json:"cmd_id"`
	CmdType     string         `gorm:"size:128" json:"cmd_type"`
	Status      string         `gorm:"size:64;default: ready" json:"status"`
	ExecuteID   int            `json:"execute_id"`
	ExecuteType string         `gorm:"size:64" json:"execute_type"`
	Instances   []TaskInstance `gorm:"constraint:OnDelete:CASCADE;" json:"instances"`
}

type TaskInstance struct {
	Id        int       `json:"id"`
	Uid       string    `json:"uid"`
	JobId     int       `json:"job_id"`
	Job       Job       `json:"-"`
	StartTime time.Time `gorm:"index" json:"start_time"`
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

func InsertJob(name, t, spec, cmd string, executeID, cmdId int, executeType, cmdType string) (*Job, error) {
	job := Job{
		Name:        name,
		Type:        t,
		Spec:        spec,
		Cmd:         cmd,
		CmdId:       cmdId,
		ExecuteID:   executeID,
		ExecuteType: executeType,
		CmdType:     cmdType,
	}
	err := db.Create(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func UpdateJob(id int, name, t, spec, cmd, cmdType string, cmdId int) (*Job, error) {
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
	if cmdType != "" {
		job.CmdType = cmdType
	}
	if cmdId != 0 {
		job.CmdId = cmdId
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
	tPath := ti.StartTime.Format("20060102")
	return path.Join(tmpPath, tPath, fmt.Sprintf("%s.log", ti.Uid))
}

func (ti *TaskInstance) UpdateStatus(status string) error {
	return db.Model(&TaskInstance{}).Where("id", ti.Id).Update("status", status).Error
}

func (ti *TaskInstance) Done() error {
	db.Model(&TaskInstance{}).Where("id", ti.Id).Update("end_time", time.Now().Local())
	return ti.UpdateStatus(InstanceStatusDone)
}

func GetTaskInstanceById(id int) (*TaskInstance, error) {
	var instance *TaskInstance
	err := db.Preload("Job").Where("id", id).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return instance, nil
}

func UpdateTaskInstanceLogTrace(instance *TaskInstance, logPath string) error {
	instance.LogPath = logPath
	return db.Model(&TaskInstance{}).Where("id", instance.Id).Update("log_path", logPath).Error
}

func InsertTaskInstance(jobId int, start time.Time) (*TaskInstance, error) {
	job, err := GetJobById(jobId)
	if err != nil {
		return nil, err
	}
	instance := TaskInstance{
		JobId:     jobId,
		StartTime: start,
		Uid:       uuid.NewString(),
	}
	err = db.Create(&instance).Error
	if err != nil {
		return nil, err
	}

	instance.Job = *job

	return &instance, nil
}

func clearJobLogs(sinceBefore time.Time, job *Job) error {
	logPath := filepath.Join(path.Join(dataPath, config.DefaultTaskTmpPath), fmt.Sprintf("%d-%s", job.Id, job.Name))
	files, err := ioutil.ReadDir(logPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		parse, err := time.Parse("20060102", f.Name())
		if err == nil {
			if parse.Before(sinceBefore) {
				_ = os.RemoveAll(filepath.Join(logPath, f.Name()))
			}
		}
	}

	return nil
}

func ClearInstance(sinceBefore time.Time, jobId int) (err error) {
	if sinceBefore.IsZero() {
		return errors.New("must have a sinceBefore")
	}

	var job *Job

	if jobId > 0 {
		job, err = GetJobById(jobId)
		if err != nil {
			return err
		}
		db.Where("job_id", jobId).Where("end_time < ?", sinceBefore).Delete(&TaskInstance{})
	} else {
		db.Where("end_time < ?", sinceBefore).Delete(&TaskInstance{})
	}

	if job != nil {
		err := clearJobLogs(sinceBefore, job)
		if err != nil {
			return err
		}
	} else {
		allJob, err := GetAllJob()
		if err != nil {
			return err
		}

		for _, job := range allJob {
			err := clearJobLogs(sinceBefore, job)
			if err != nil {
				continue
			}
		}
	}

	return nil
}
