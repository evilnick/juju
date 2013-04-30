package testing

import (
	"fmt"
	"go/build"
	"launchpad.net/juju-core/charm"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	p, err := build.Import("launchpad.net/juju-core/testing", "", build.FindOnly)
	check(err)
	Charms = &Repo{Path: filepath.Join(p.Dir, "repo")}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Repo represents a charm repository used for testing.
type Repo struct {
	Path string
}

// Charms represents the specific charm repository stored in this package and
// used by the Juju unit tests. The series name is "series".
var Charms *Repo

func clone(dst, src string) string {
	check(exec.Command("cp", "-r", src, dst).Run())
	return filepath.Join(dst, filepath.Base(src))
}

// DirPath returns the path to a charm directory with the given name in the
// default series
func (r *Repo) DirPath(name string) string {
	return filepath.Join(r.Path, "series", name)
}

// Dir returns the actual charm.Dir named name.
func (r *Repo) Dir(name string) *charm.Dir {
	ch, err := charm.ReadDir(r.DirPath(name))
	check(err)
	return ch
}

// ClonedDirPath returns the path to a new copy of the default charm directory
// named name.
func (r *Repo) ClonedDirPath(dst, name string) string {
	return clone(dst, r.DirPath(name))
}

// RenamedClonedDirPath returns the path to a new copy of the default
// charm directory named name, but renames it to newName.
func (r *Repo) RenamedClonedDirPath(dst, name, newName string) string {
	newDst := clone(dst, r.DirPath(name))
	renamedDst := filepath.Join(filepath.Dir(newDst), newName)
	check(os.Rename(newDst, renamedDst))
	return renamedDst
}

// ClonedDir returns an actual charm.Dir based on a new copy of the charm directory
// named name, in the directory dst.
func (r *Repo) ClonedDir(dst, name string) *charm.Dir {
	ch, err := charm.ReadDir(r.ClonedDirPath(dst, name))
	check(err)
	return ch
}

// ClonedURL makes a copy of the charm directory. It will create a directory
// with the series name if it does not exist, and then clone the charm named
// name into that directory. The return value is a URL pointing at the local
// charm.
func (r *Repo) ClonedURL(dst, series, name string) *charm.URL {
	dst = filepath.Join(dst, series)
	if err := os.MkdirAll(dst, 0777); err != nil {
		panic(fmt.Errorf("cannot make destination directory: %v", err))
	}
	clone(dst, r.DirPath(name))
	return &charm.URL{
		Schema:   "local",
		Series:   series,
		Name:     name,
		Revision: -1,
	}
}

// BundlePath returns the path to a new charm bundle file created from the
// charm directory named name, in the directory dst.
func (r *Repo) BundlePath(dst, name string) string {
	dir := r.Dir(name)
	path := filepath.Join(dst, "bundle.charm")
	file, err := os.Create(path)
	check(err)
	defer file.Close()
	check(dir.BundleTo(file))
	return path
}

// Bundle returns an actual charm.Bundle created from a new charm bundle file
// created from the charm directory named name, in the directory dst.
func (r *Repo) Bundle(dst, name string) *charm.Bundle {
	ch, err := charm.ReadBundle(r.BundlePath(dst, name))
	check(err)
	return ch
}
