package integration_test

import (
	"fmt"
	"github.com/cloudfoundry/libbuildpack"
	"os/exec"
	"path/filepath"
)

func runCNB2CF(dir string, args ...string) (string, error) {
	rootDir, err := FindRoot()
	if err != nil {
		return "", err
	}
	binary := filepath.Join(rootDir, "build", "cnb2cf")
	cmd := exec.Command(binary, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func FindRoot() (string, error) {
	dir, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	for {
		if dir == "/" {
			return "", fmt.Errorf("Could not find README.md in the directory hierarchy")
		}
		if exist, err := libbuildpack.FileExists(filepath.Join(dir, "README.md")); err != nil {
			return "", err
		} else if exist {
			return dir, nil
		}
		dir, err = filepath.Abs(filepath.Join(dir, ".."))
		if err != nil {
			return "", err
		}
	}
}
