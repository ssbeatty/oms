package schedule

import (
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"sync"
)

type Schedule struct {
	inner *cron.Cron
	ids   map[string]cron.EntryID
	mutex sync.Mutex
}

func (s *Schedule) IDs() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	validIDs := make([]string, 0, len(s.ids))
	invalidIDs := make([]string, 0)
	for sid, eid := range s.ids {
		if e := s.inner.Entry(eid); e.ID != eid {
			invalidIDs = append(invalidIDs, sid)
			continue
		}
		validIDs = append(validIDs, sid)
	}
	for _, id := range invalidIDs {
		delete(s.ids, id)
	}
	return validIDs
}

func (s *Schedule) Start() {
	s.inner.Start()
}

func (s *Schedule) Close() {
	s.inner.Stop()
}

func (s *Schedule) AddByJob(id string, spec string, cmd cron.Job) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.ids[id]; ok {
		return errors.Errorf("crontab id exists")
	}
	eid, err := s.inner.AddJob(spec, cmd)
	if err != nil {
		return err
	}
	s.ids[id] = eid
	return nil
}

func (s *Schedule) AddByFunc(id string, spec string, f func()) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.ids[id]; ok {
		return errors.Errorf("crontab id exists")
	}
	eid, err := s.inner.AddFunc(spec, f)
	if err != nil {
		return err
	}
	s.ids[id] = eid
	return nil
}

func (s *Schedule) IsExists(jid string) bool {
	_, exist := s.ids[jid]
	return exist
}

func (s *Schedule) Remove(id string) {
	if s.IsExists(id) {
		s.inner.Remove(s.ids[id])
		delete(s.ids, id)
	}
}

func NewSchedule() *Schedule {
	return &Schedule{
		inner: cron.New(cron.WithSeconds()),
		ids:   make(map[string]cron.EntryID),
	}
}
