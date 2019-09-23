package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type ManifestYAML struct {
	Language     string       `yaml:"language"`
	PrePackage   string       `yaml:"pre_package"`
	IncludeFiles []string     `yaml:"include_files"`
	Dependencies []Dependency `yaml:"dependencies"`
	Stack        string       `yaml:"stack"`
}

type Dependency struct {
	Name         string   `toml:"name" yaml:"name"`
	ID           string   `toml:"id"`
	SHA256       string   `toml:"sha256" yaml:"sha256"`
	Source       string   `toml:"source,omitempty" yaml:"source,omitempty"`
	SourceSHA256 string   `toml:"source_sha256" yaml:"source_sha256,omitempty"`
	CFStacks     []string `toml:"stacks" yaml:"cf_stacks"`
	URI          string   `toml:"uri" yaml:"uri"`
	Version      string   `toml:"version" yaml:"version"`
}

const Lifecycle = "lifecycle"

func UpdateDependency(dependency Dependency, path string) (Dependency, error) {
	dependency.URI = fmt.Sprintf("file://%s", path)
	sha, err := getSHA256(path)
	if err != nil {
		return Dependency{}, err
	}

	dependency.SHA256 = hex.EncodeToString(sha[:])

	for i, stack := range dependency.CFStacks {
		// Translate stack from org.cloudfoundry.stacks.cflinuxfs3 to just cflinuxfs3
		dependency.CFStacks[i] = strings.Split(stack, ".stacks.")[1]
	}

	dependency.Name = dependency.ID

	return dependency, nil
}

func (m *ManifestYAML) Initialize(language string) {
	m.IncludeFiles = []string{
		"bin/compile",
		"bin/detect",
		"bin/finalize",
		"bin/release",
		"bin/supply",
		"buildpack.toml",
		"manifest.yml",
		"VERSION",
	}
	splitLanguage := strings.Split(language, ".")
	m.Language = splitLanguage[len(splitLanguage)-1]
}

func (m *ManifestYAML) Write(path string) error {
	contents, err := yaml.Marshal(&m)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, contents, 0666)
}

func getSHA256(path string) ([32]byte, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}

	return sha256.Sum256(content), nil
}
