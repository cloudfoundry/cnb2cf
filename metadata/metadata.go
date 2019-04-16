package metadata

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type BuildpackTOML struct {
	Buildpack CNBBuildpack           `toml:"buildpack"`
	Metadata  map[string]interface{} `toml:"metadata"`
	Stacks    []struct {
		ID          string   `toml:"id"`
		Mixins      []string `toml:"mixins"`
		BuildImages []string `toml:"build-images"`
		RunImages   []string `toml:"run-images"`
	} `toml:"stacks"`
}

type OrderTOML struct {
	Groups []CNBGroup `toml:"groups" yaml:"groups"`
}

type CNBGroup struct {
	Buildpacks []CNBBuildpack `toml:"buildpacks" yaml:"buildpacks"`
}

type CNBBuildpack struct {
	ID       string `toml:"id" yaml:"id"`
	Name     string `toml:"name,omitempty"`
	Version  string `toml:"version" yaml:"version"`
	Optional bool   `toml:"optional,omitempty" yaml:"optional,omitempty"`
}

type ManifestYAML struct {
	Language     string         `yaml:"language"`
	PrePackage   string         `yaml:"pre_package"`
	IncludeFiles []string       `yaml:"include_files"`
	Dependencies []V2Dependency `yaml:"dependencies"`
	Stack        string         `yaml:"stack"`
}

func (m *ManifestYAML) Load(path string) error {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(contents, m)
}

func (m *ManifestYAML) Write(path string) error {
	contents, err := yaml.Marshal(&m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, contents, 0666)
}

type V2Dependency struct {
	Name         string   `yaml:"name"`
	Version      string   `yaml:"version"`
	URI          string   `yaml:"uri"`
	SHA256       string   `yaml:"sha256"`
	CFStacks     []string `yaml:"cf_stacks"`
	Source       string   `yaml:"source,omitempty"`
	SourceSHA256 string   `yaml:"source_sha256,omitempty"`
}
