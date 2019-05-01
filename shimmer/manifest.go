package shimmer

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type ManifestYAML struct {
	Language     string         `yaml:"language"`
	PrePackage   string         `yaml:"pre_package"`
	IncludeFiles []string       `yaml:"include_files"`
	Dependencies []V2Dependency `yaml:"dependencies"`
	Stack        string         `yaml:"stack"`
}

func LoadManifest() (ManifestYAML, error) {
	contents, err := ioutil.ReadFile("manifest.yml")
	if err != nil {
		return ManifestYAML{}, err
	}

	manifest := ManifestYAML{}
	if err := yaml.Unmarshal(contents, &manifest); err != nil {
		return ManifestYAML{}, err
	}


	return manifest, nil
}
