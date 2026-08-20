package validate

import (
	"testing"
	"time"
)

const maxAge = 9 * 24 * time.Hour

func nameFor(date time.Time) string {
	return "LADD_Industry_Filter_CUI_SP_PRVCY_" + date.Format("20060102") + ".txt"
}

func TestCheckAcceptsValid(t *testing.T) {
	content := []byte("A0B1C2\n# a comment\nD3E4F5\n\nN12345\n")
	res, err := Check(nameFor(time.Now()), content, maxAge)
	if err != nil {
		t.Fatalf("valid file rejected: %v", err)
	}
	if res.Entries != 3 {
		t.Fatalf("entries = %d, want 3", res.Entries)
	}
}

func TestCheckRejects(t *testing.T) {
	stale := time.Now().AddDate(0, 0, -30)
	future := time.Now().AddDate(0, 0, 2)

	cases := []struct {
		name    string
		file    string
		content string
	}{
		{"wrong prefix", "random_file_20260101.txt", "A0B1C2\n"},
		{"no date", "LADD_Industry_Filter_CUI.txt", "A0B1C2\n"},
		{"stale", nameFor(stale), "A0B1C2\n"},
		{"future dated", nameFor(future), "A0B1C2\n"},
		{"no entries", nameFor(time.Now()), "# only comments\n\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Check(c.file, []byte(c.content), maxAge); err == nil {
				t.Fatal("expected rejection, got nil error")
			}
		})
	}
}

func TestCheckStripsPath(t *testing.T) {
	res, err := Check("../../etc/"+nameFor(time.Now()), []byte("A0B1C2\n"), maxAge)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Name != nameFor(time.Now()) {
		t.Fatalf("name = %q, want the base name", res.Name)
	}
}
