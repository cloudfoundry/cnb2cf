package creator

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/cnb2cf/utils"

	"github.com/cloudfoundry/cnb2cf/metadata"

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

func CreateZip(config Config, srcDir, outputDir string) error {
	bpZip := filepath.Join(outputDir, fmt.Sprintf("%s_buildpack-%s-%s.zip", config.Language, config.Stack, config.Version))
	return zipFiles(srcDir, bpZip)
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

		return utils.WriteToFile(srcFile, filepath.Join(outputDir, path), 0777)
	}); err != nil {
		return err
	}

	return nil
}

func updateManifest(cfg Config, file io.Reader) (io.Reader, error) {
	manifest := metadata.ManifestYAML{}
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
	orderTOML := metadata.OrderTOML{
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

func zipFiles(srcDir, filename string) error {
	newfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newfile.Close()

	zipWriter := zip.NewWriter(newfile)
	defer zipWriter.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.Join(".", strings.TrimPrefix(path, srcDir))
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})
}