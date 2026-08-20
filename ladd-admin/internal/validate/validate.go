package validate

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var dateInName = regexp.MustCompile(`(\d{8})\.txt$`)

type Result struct {
	Name    string
	Date    time.Time
	Entries int
}

func Check(filename string, content []byte, maxAge time.Duration) (Result, error) {
	name := baseName(filename)

	if !strings.HasPrefix(name, "LADD_Industry_Filter") {
		return Result{}, fmt.Errorf("unexpected filename %q (want LADD_Industry_Filter_*_YYYYMMDD.txt)", name)
	}

	date, ok := publishDate(name)
	if !ok {
		return Result{}, fmt.Errorf("no YYYYMMDD date in filename %q", name)
	}

	now := time.Now()
	if date.After(now) {
		return Result{}, fmt.Errorf("file is future-dated (%s)", date.Format("2006-01-02"))
	}
	if maxAge > 0 && now.Sub(date) > maxAge {
		return Result{}, fmt.Errorf("file is stale (published %s, older than %s)", date.Format("2006-01-02"), maxAge)
	}

	entries := countEntries(content)
	if entries == 0 {
		return Result{}, fmt.Errorf("file has no entries")
	}

	return Result{Name: name, Date: date, Entries: entries}, nil
}

func baseName(name string) string {
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	return name
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

func countEntries(content []byte) int {
	n := 0
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		n++
	}
	return n
}
