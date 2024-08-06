package streamer

import (
	"archive/tar"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// Tar helper to create a tar instance based on a source directory, skipping entries that are not
// desired like `.git` directory and entries in `.gitignore` file.
type Tar struct {
	src       string            // base directory
	gitIgnore *ignore.GitIgnore // matcher for git ignored files
}

// skipPath inspect each path and makes sure it skips files the tar helper can't handle.
func (t *Tar) skipPath(fpath string, stat fs.FileInfo) bool {
	if !stat.Mode().IsRegular() {
		return true
	}
	if strings.HasPrefix(fpath, path.Join(t.src, ".git")) {
		return true
	}
	if t.gitIgnore == nil {
		return false
	}
	return t.gitIgnore.MatchesPath(fpath)
}

// Create the actual tar by inspecting all files in source path, skipping some.
func (t *Tar) Create(w io.Writer) error {
	tw := tar.NewWriter(w)
	if err := filepath.Walk(t.src, func(fpath string, stat fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if t.skipPath(fpath, stat) {
			return nil
		}
		return writeFileToTar(tw, t.src, fpath, stat)
	}); err != nil {
		return err
	}
	return tw.Close()
}

// bootstrap instantiate git-ignore helper.
func (t *Tar) bootstrap() error {
	gitIgnorePath := path.Join(t.src, ".gitignore")
	_, err := os.Stat(gitIgnorePath)
	if err != nil {
		return nil
	}

	t.gitIgnore, err = ignore.CompileIgnoreFile(gitIgnorePath)
	return err
}

// NewTar instantiate a tar helper based on the source directory path informed.
func NewTar(src string) (*Tar, error) {
	t := &Tar{src: src}
	return t, t.bootstrap()
}

// Size returns the size in bytes that a call of Create would write into the io.Writer
//
// Note: This performs a whote walk through of the source directory and can be an expensive operation.
func (t *Tar) Size() (int, error) {
	wc := &writeCounter{}
	if err := t.Create(wc); err != nil {
		return -1, err
	}

	return wc.total, nil
}
