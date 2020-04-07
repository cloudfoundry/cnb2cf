package shims

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/cnb2cf/cloudnative"

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

func WritePlatformDir(platformDir string, envs []string) error {
	envDir := filepath.Join(platformDir, "env")
	err := os.MkdirAll(envDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to make env dir: %s", err)
	}

	for _, en := range envs {
		pair := strings.SplitN(en, "=", 2)
		if len(pair) != 2 {
			return fmt.Errorf("var fails to contain required key=value structure")
		}
		key := pair[0]
		val := pair[1]
		err = ioutil.WriteFile(filepath.Join(envDir, key), []byte(val), os.ModePerm)
		if err != nil {
			return fmt.Errorf("unable to write %s env file: %s", key, err)
		}
	}
	return nil
}

// See https://github.com/buildpacks/rfcs/blob/40babff3e4c062ebb00e669ee50ca649e9b81944/text/0022-client-side-buildpack-registry.md#how-it-works
// "Note: id is the combination of two fields, ns and name. The / will be replaced by a _ in the filename"
func SanitizeId(id string) string {
	return strings.ReplaceAll(id, "/", "_")
}
