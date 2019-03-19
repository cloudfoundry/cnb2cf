package shimmer

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	_ "github.com/cloudfoundry/cnb2cf/statik"
	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"
	"gopkg.in/yaml.v2"
)

func CreateBuildpack(cfg Config, outputDir string) error {
	if err := copyTemplate(cfg, outputDir); err != nil {
		return errors.Wrap(err, "failed to copy template")
	}

	if err := writeVersion(cfg.Version, outputDir); err != nil {
		return errors.Wrap(err, "failed to write VERSION")
	}

	return generateOrderTOML(cfg, outputDir)
}

func writeVersion(version, outputDir string) error {
	return ioutil.WriteFile(filepath.Join(outputDir, "VERSION"), []byte(version), 0666)
}

func copyTemplate(cfg Config, outputDir string) error {
	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	if err := fs.Walk(statikFS, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := statikFS.Open(path)
		if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return err
		}
		defer file.Close()

		var srcFile io.Reader = file

		if info.Name() == "manifest.yml" {
			srcFile, err = updateManifest(cfg, srcFile)
		}

		return writeToFile(srcFile, filepath.Join(outputDir, path), 0777)
	}); err != nil {
		return err
	}

	return nil
}

func updateManifest(cfg Config, file io.Reader) (io.Reader, error) {
	manifest := ManifestYAML{}
	if err := yaml.NewDecoder(file).Decode(&manifest); err != nil {
		return nil, err
	}
	manifest.Language = cfg.Language
	manifest.Stack = cfg.Stack
	manifest.Dependencies = append(manifest.Dependencies, cfg.Buildpacks...)

	contents, err := yaml.Marshal(&manifest)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(contents), nil
}

func generateOrderTOML(cfg Config, outputDir string) error {
	orderTOML := OrderTOML{
		Groups: cfg.Groups,
	}

	for i, group := range orderTOML.Groups {
		for j := range group.Buildpacks {
			orderTOML.Groups[i].Buildpacks[j].Version = "latest"
		}
	}

	orderTOMLFile, err := os.Create(filepath.Join(outputDir, "order.toml"))
	if err != nil {
		return err
	}
	defer orderTOMLFile.Close()

	return toml.NewEncoder(orderTOMLFile).Encode(orderTOML)
}

func writeToFile(source io.Reader, destFile string, mode os.FileMode) error {
	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return err
	}

	fh, err := os.OpenFile(destFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = io.Copy(fh, source)
	if err != nil {
		return err
	}

	return nil
}
