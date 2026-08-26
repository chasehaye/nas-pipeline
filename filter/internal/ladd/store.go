package ladd

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type Store struct {
	current atomic.Pointer[snapshot]
	maxAge  time.Duration
}

type snapshot struct {
	set           *Set
	effectiveDate time.Time
}

func NewStore(maxAge time.Duration) *Store {
	s := &Store{maxAge: maxAge}
	s.current.Store(&snapshot{set: NewSet(nil)})
	return s
}

func (s *Store) Swap(set *Set, effectiveDate time.Time) {
	s.current.Store(&snapshot{set: set, effectiveDate: effectiveDate})
}

func (s *Store) Reload(d Dirs) (swapped bool, err error) {
	_, perr := Promote(d)

	set, date, lerr := LoadLatest(d.Active)
	if lerr != nil {
		return false, errors.Join(perr, lerr)
	}
	if !date.After(s.current.Load().effectiveDate) {
		return false, perr
	}
	s.Swap(set, date)
	return true, perr
}

func (s *Store) Blocks(callSign, registration string) bool {
	return s.current.Load().set.Blocks(callSign, registration)
}

func (s *Store) Ready() (bool, string) {
	snap := s.current.Load()
	if snap.effectiveDate.IsZero() || snap.set.Len() == 0 {
		return false, "no LADD list loaded"
	}
	if s.maxAge > 0 && time.Since(snap.effectiveDate) > s.maxAge {
		return false, fmt.Sprintf("LADD list is stale (published %s, older than %s)",
			snap.effectiveDate.Format("2006-01-02"), s.maxAge)
	}
	return true, ""
}
