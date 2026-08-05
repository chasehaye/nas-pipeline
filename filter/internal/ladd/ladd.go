package ladd

import "strings"


type Set struct {
	entries map[string]struct{}
}


func NewSet(ids []string) *Set {
	m := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if k := normalize(id); k != "" {
			m[k] = struct{}{}
		}
	}
	return &Set{entries: m}
}

func (s *Set) Len() int {
	if s == nil {
		return 0
	}
	return len(s.entries)
}

func (s *Set) Blocks(callSign, registration string) bool {
	if s == nil {
		return false
	}
	return s.has(callSign) || s.has(registration)
}

func (s *Set) has(id string) bool {
	k := normalize(id)
	if k == "" {
		return false
	}
	_, ok := s.entries[k]
	return ok
}

func normalize(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
