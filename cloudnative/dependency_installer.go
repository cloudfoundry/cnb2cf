package cloudnative

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/packager"
)

type DependencyInstaller struct{}

func NewDependencyInstaller() DependencyInstaller {
	return DependencyInstaller{}
}

func (di DependencyInstaller) Download(uri, checksum, destination string) error {
	if err := packager.DownloadFromURI(uri, destination); err != nil {
		return err
	}

	if err := libbuildpack.CheckSha256(destination, checksum); err != nil {
		return err
	}

	return nil
}

func (di DependencyInstaller) Copy(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("source %s is not a directory", source)
	}

	if err := os.MkdirAll(destination, 0755); err != nil {
		return err
	}

	if err := libbuildpack.CopyDirectory(source, destination); err != nil {
		return err
	}

	return nil
}
