package cloudnative

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

//go:generate faux --interface filesystem --output fakes/filesystem.go
type filesystem interface {
	ReadFile(name string) ([]byte, error)
}

type LifecycleHooks struct {
	fs filesystem
}

func NewLifecycleHooks(fs filesystem) LifecycleHooks {
	return LifecycleHooks{
		fs: fs,
	}
}

func (lh LifecycleHooks) Install(name, directory string) error {
	contents, err := lh.fs.ReadFile(filepath.Join("/bin", name))
	if err != nil {
		return fmt.Errorf("could not install hook %q: %s", name, err)
	}

	binDir := filepath.Join(directory, "bin")
	if err := os.MkdirAll(binDir, os.ModePerm); err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(binDir, name), contents, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
