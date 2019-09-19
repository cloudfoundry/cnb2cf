package shims

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/buildpack/libbuildpack/buildpack"

	buildpack2 "github.com/buildpack/libbuildpack/buildpack"
)

type Order struct {
	Groups []buildpack2.Info `toml:"group"`
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
		orderTOML, err := ParseBuildpackTOML(filepath.Join(orderFilesDir, file.Name()))
		if err != nil {
			return err
		}

		*orders = append(*orders, orderTOML...)
	}

	return nil
}

func ParseBuildpackTOML(path string) ([]Order, error) {
	var buildpack struct {
		Orders []Order `toml:"order"`
	}
	if _, err := toml.DecodeFile(path, &buildpack); err != nil {
		return nil, err
	}

	return buildpack.Orders, nil
}

func encodeTOML(dest string, data interface{}) error {
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	return toml.NewEncoder(destFile).Encode(data)
}
