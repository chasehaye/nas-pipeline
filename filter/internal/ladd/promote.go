package ladd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Dirs are the three directories that make up the LADD list lifecycle:
//
//	Staging  - where a freshly delivered file lands (written, then renamed in).
//	Active   - the single directory the loader reads from; newest date wins.
//	Archived - superseded files are moved here and kept indefinitely for audit.
type Dirs struct {
	Staging  string
	Active   string
	Archived string
}

func Promote(d Dirs) (promoted string, err error) {
	stagedName, stagedDate, err := newestDated(d.Staging)
	if err != nil {
		return "", fmt.Errorf("scanning staging %s: %w", d.Staging, err)
	}
	if stagedName == "" {
		return "", nil
	}

	_, activeDate, err := newestDated(d.Active)
	if err != nil {
		return "", fmt.Errorf("scanning active %s: %w", d.Active, err)
	}

	if !stagedDate.After(activeDate) {
		_ = moveInto(d.Archived, filepath.Join(d.Staging, stagedName))
		return "", nil
	}

	if err := moveInto(d.Active, filepath.Join(d.Staging, stagedName)); err != nil {
		return "", fmt.Errorf("promoting %s to active: %w", stagedName, err)
	}

	archiveLADD(d.Active, d.Archived, stagedName)
	archiveLADD(d.Staging, d.Archived, "")

	return stagedName, nil
}

func archiveLADD(dir, archiveDir, keep string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == keep {
			continue
		}
		if !strings.HasPrefix(e.Name(), "LADD_Industry_Filter") {
			continue
		}
		_ = moveInto(archiveDir, filepath.Join(dir, e.Name()))
	}
}

func moveInto(destDir, srcPath string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	dest := uniquePath(filepath.Join(destDir, filepath.Base(srcPath)))

	if err := os.Rename(srcPath, dest); err == nil {
		return nil
	}
	if err := copyFile(srcPath, dest); err != nil {
		return err
	}
	return os.Remove(srcPath)
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		cand := fmt.Sprintf("%s.%d%s", stem, i, ext)
		if _, err := os.Stat(cand); errors.Is(err, os.ErrNotExist) {
			return cand
		}
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
