package shims

import (
	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/buildpack/libbuildpack/buildpack"
)

// need to add optional stuff here


type Order struct {
	Groups []cloudnative.BuildpackOrderGroup `toml:"group"`
}

type BuildpackTOML struct {
	buildpack.Buildpack
	Order []Order `toml:"order"`
}

func parseOrderTOMLs(orders *[]Order, orderFilesDir string) error {
	orderFiles, err := ioutil.ReadDir(orderFilesDir)
	if err != nil {
		return err
	}

	for _, file := range orderFiles {
		buildpack, err := ParseBuildpackTOML(filepath.Join(orderFilesDir, file.Name()))
		if err != nil {
			return err
		}

		*orders = append(*orders, buildpack.Order...)
	}

	return nil
}

func ParseBuildpackTOML(path string) (BuildpackTOML, error) {
	var buildpack BuildpackTOML
	if _, err := toml.DecodeFile(path, &buildpack); err != nil {
		return BuildpackTOML{}, err
	}

	return buildpack, nil
}

func encodeTOML(dest string, data interface{}) error {
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	return toml.NewEncoder(destFile).Encode(data)
}
