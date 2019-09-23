package cloudnative

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Manifest struct {
	Language     string               `yaml:"language"`
	IncludeFiles []string             `yaml:"include_files"`
	Dependencies []ManifestDependency `yaml:"dependencies"`
}
type ManifestDependency struct {
	Name         string   `yaml:"name"`
	ID           string   `yaml:"id"`
	SHA256       string   `yaml:"sha256"`
	Stacks       []string `yaml:"cf_stacks"`
	URI          string   `yaml:"uri"`
	Version      string   `yaml:"version"`
	Source       string   `yaml:"source"`
	SourceSHA256 string   `yaml:"source_sha256"`
}

func WriteManifest(manifest Manifest, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	err = yaml.NewEncoder(file).Encode(manifest)
	if err != nil {
		return err
	}

	return nil
}
