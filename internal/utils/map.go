package utils

import (
	"sync"
	"sync/atomic"
)

type SafeMap struct {
	inner *sync.Map
	len   int64
}

func NewSafeMap() *SafeMap {
	return &SafeMap{
		inner: &sync.Map{},
	}
}

func (s *SafeMap) Load(key interface{}) (value interface{}, ok bool) {
	return s.inner.Load(key)
}

func (s *SafeMap) Store(key, value interface{}) {
	atomic.AddInt64(&s.len, 1)
	s.inner.Store(key, value)
}

func (s *SafeMap) Delete(key interface{}) {
	if _, ok := s.inner.LoadAndDelete(key); ok {
		atomic.AddInt64(&s.len, -1)
	}
}

func (s *SafeMap) Range(f func(key, value interface{}) bool) {
	s.inner.Range(f)
}

func (s *SafeMap) Length() int {
	return int(s.len)
}
