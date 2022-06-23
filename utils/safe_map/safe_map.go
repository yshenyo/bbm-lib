package global

import (
	"errors"
	"sync"
)

var ErrSafeMapLoad = errors.New("safe map load error")

// SafeMap taskid:[total,success,fail]
type SafeMap struct {
	SafeMap *sync.Map
	mux     sync.Mutex
}

func NewSafeMap() *SafeMap {
	var s sync.Map
	return &SafeMap{
		SafeMap: &s,
	}
}

func (s *SafeMap) Store(key interface{}, value interface{}) {
	s.SafeMap.Store(key, value)
}

func (s *SafeMap) Delete(key interface{}) {
	s.SafeMap.Delete(key)
}

func (s *SafeMap) Load(key interface{}) (value interface{}, ok bool) {
	value, ok = s.SafeMap.Load(key)
	if !ok {
		value, ok = s.SafeMap.Load(key)
		if !ok {
			value, ok = s.SafeMap.Load(key)
		}
	}
	return value, ok
}

func (s *SafeMap) LoadOrStore(key interface{}, value interface{}) (actual interface{}, loaded bool) {
	return s.SafeMap.LoadOrStore(key, value)
}

func (s *SafeMap) AddTotalSyncMap(key interface{}, total int) (err error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	value, ok := s.Load(key)
	if !ok {
		return ErrSafeMapLoad
	}
	arr := value.([]int)
	arr[0] = arr[0] + total
	s.Store(key, arr)
	return nil
}

func (s *SafeMap) AddSuccessSyncMap(key interface{}, success int) (err error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	value, ok := s.Load(key)
	if !ok {
		return ErrSafeMapLoad
	}
	arr := value.([]int)
	arr[1] = arr[1] + success
	s.Store(key, arr)
	return nil
}

func (s *SafeMap) AddFailSyncMap(key interface{}, fail int) (err error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	value, ok := s.Load(key)
	if !ok {
		return ErrSafeMapLoad
	}
	arr := value.([]int)
	arr[2] = arr[2] + fail
	s.Store(key, arr)
	return nil
}

func (s *SafeMap) CheckFinish(key interface{}) (isFinish bool, cut []int, err error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	value, ok := s.Load(key)
	if !ok {
		return false, cut, ErrSafeMapLoad
	}
	arr := value.([]int)
	total := arr[0]
	success := arr[1]
	fail := arr[2]
	if total == (success + fail) {
		return true, arr, nil
	} else {
		return false, arr, nil
	}
}

func (s *SafeMap) Reset(key string) {
	s.Store(key, []int{0, 0, 0})
}

func (s *SafeMap) GetCutProcess(key interface{}) (cut []int, err error) {
	value, ok := s.Load(key)
	if !ok {
		return cut, ErrSafeMapLoad
	}
	cut = value.([]int)
	return cut, nil
}
