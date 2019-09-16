package metadata

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"

	"github.com/BurntSushi/toml"

	"github.com/buildpack/libbuildpack/buildpack"

	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

type BuildpackToml struct {
	API      string                 `toml:"api"`
	Info     buildpack.Info         `toml:"buildpack" validate:"required"`
	Metadata OrderBuildpackMetadata `toml:"metadata" validate:"required"`
	Order    []Order                `toml:"order" validate:"required"`
}

type OrderBuildpackMetadata struct {
	IncludeFiles []string     `toml:"include_files"`
	Dependencies []Dependency `toml:"dependencies" validate:"required"`
}

type Order struct {
	Group []CNBBuildpack `toml:"group"`
}

type CNBBuildpack struct {
	ID      string `toml:"id" yaml:"id" validate:"required"`
	Version string `toml:"version" yaml:"version" validate:"required"`
}

type ManifestYAML struct {
	Language     string       `yaml:"language"`
	PrePackage   string       `yaml:"pre_package"`
	IncludeFiles []string     `yaml:"include_files"`
	Dependencies []Dependency `yaml:"dependencies"`
	Stack        string       `yaml:"stack"`
}

type Dependency struct {
	Name         string   `toml:"name" yaml:"name"`
	ID           string   `toml:"id" validate:"required"`
	SHA256       string   `toml:"sha256" yaml:"sha256"`
	Source       string   `toml:"source,omitempty" yaml:"source,omitempty" validate:"required"`
	SourceSHA256 string   `toml:"source_sha256" yaml:"source_sha256,omitempty"`
	CFStacks     []string `toml:"stacks" yaml:"cf_stacks"`
	URI          string   `toml:"uri" yaml:"uri"`
	Version      string   `toml:"version" yaml:"version"`
}

const Lifecycle = "lifecycle"

func (d *Dependency) UpdateDependency(depPath string) error {
	d.URI = fmt.Sprintf("file://%s", depPath)
	sha, err := getSHA256(depPath)
	if err != nil {
		return err
	}

	d.SHA256 = hex.EncodeToString(sha[:])

	for i, stack := range d.CFStacks {
		// Translate stack from org.cloudfoundry.stacks.cflinuxfs3 to just cflinuxfs3
		d.CFStacks[i] = strings.Split(stack, ".stacks.")[1]
	}

	d.Name = d.ID
	return nil
}

func (obp *BuildpackToml) Load(path string) error {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrap(err, "failed to read the buildpack.toml")
	}

	if _, err = toml.Decode(string(contents), obp); err != nil {
		return errors.Wrap(err, "failed to decode the buildpack.toml")
	}

	return obp.Validate()
}

func (obp *BuildpackToml) Validate() error {
	validate := validator.New()
	err := validate.Struct(obp)
	if err != nil {
		return errors.Wrap(err, "failed to validate buildpack.toml")
	}

	err = validate.Struct(obp.Metadata)
	if err != nil {
		return errors.Wrap(err, "failed to validate buildpack.toml's metadata")
	}

	dependenciesSet := map[string]string{}
	for _, dep := range obp.Metadata.Dependencies {
		dependenciesSet[dep.ID] = dep.Version
		if dep.Source == "" {
			return fmt.Errorf("must provide a source for the dependencies")
		}
	}

	if _, ok := dependenciesSet[Lifecycle]; !ok {
		return fmt.Errorf("you must include a lifecycle in the dependencies")
	}

	for _, group := range obp.Order {
		for _, entry := range group.Group {
			if dependenciesSet[entry.ID] != entry.Version {
				return fmt.Errorf("group entry %s with version %s is not a dependency", entry.ID, entry.Version)
			}
		}
	}

	return nil
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
