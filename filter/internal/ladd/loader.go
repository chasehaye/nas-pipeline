package ladd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var dateInName = regexp.MustCompile(`(\d{8})\.txt$`)

func LoadLatest(dir string) (*Set, time.Time, error) {
	name, date, err := newestDated(dir)
	if err != nil {
		return nil, time.Time{}, err
	}
	if name == "" {
		return nil, time.Time{}, fmt.Errorf("no LADD_Industry_Filter_*.txt found in %s", dir)
	}

	set, err := loadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, time.Time{}, err
	}
	return set, date, nil
}

func newestDated(dir string) (name string, date time.Time, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", time.Time{}, nil
		}
		return "", time.Time{}, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasPrefix(n, "LADD_Industry_Filter") {
			continue
		}
		d, ok := publishDate(n)
		if !ok {
			continue
		}
		if d.After(date) {
			date = d
			name = n
		}
	}
	return name, date, nil
}

func publishDate(name string) (time.Time, bool) {
	m := dateInName.FindStringSubmatch(name)
	if m == nil {
		return time.Time{}, false
	}
	d, err := time.Parse("20060102", m[1])
	if err != nil {
		return time.Time{}, false
	}
	return d, true
}

func loadFile(path string) (*Set, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var ids []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ids = append(ids, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return NewSet(ids), nil
}
