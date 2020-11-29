package tasks

import (
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"log"
	"sync"
)

type TaskServices struct {
	inner *cron.Cron
	ids   map[string]cron.EntryID
	mutex sync.Mutex
}

func (ts *TaskServices) IDs() []string {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()
	validIDs := make([]string, 0, len(ts.ids))
	invalidIDs := make([]string, 0)
	for sid, eid := range ts.ids {
		if e := ts.inner.Entry(eid); e.ID != eid {
			invalidIDs = append(invalidIDs, sid)
			continue
		}
		validIDs = append(validIDs, sid)
	}
	for _, id := range invalidIDs {
		delete(ts.ids, id)
	}
	return validIDs
}

func (ts *TaskServices) Start() {
	ts.inner.Start()
	ts.Init()
}

func (ts *TaskServices) Stop() {
	ts.inner.Stop()
}

func (ts *TaskServices) Init() {
	if err := ts.AddByFunc("loop-status", "*/5 * * * *", GetHostStatus); err != nil {
		log.Println("init loop-status error!", err)
	}

}

func (ts *TaskServices) AddByID(id string, spec string, cmd cron.Job) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if _, ok := ts.ids[id]; ok {
		return errors.Errorf("crontab id exists")
	}
	eid, err := ts.inner.AddJob(spec, cmd)
	if err != nil {
		return err
	}
	ts.ids[id] = eid
	return nil
}

func (ts *TaskServices) AddByFunc(id string, spec string, f func()) error {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if _, ok := ts.ids[id]; ok {
		return errors.Errorf("crontab id exists")
	}
	eid, err := ts.inner.AddFunc(spec, f)
	if err != nil {
		return err
	}
	ts.ids[id] = eid
	return nil
}

func (ts *TaskServices) IsExists(jid string) bool {
	_, exist := ts.ids[jid]
	return exist
}

func NewTaskService() *TaskServices {
	return &TaskServices{
		inner: cron.New(),
		ids:   make(map[string]cron.EntryID),
	}
}
