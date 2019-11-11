package cloudnative

import (
	"os"
	"strings"

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

func UpdateStacks(stacks []string) []string {
	var newStacks []string
	for _, stack := range stacks {
		updatedStack := strings.Split(stack, ".")
		newStacks = append(newStacks, updatedStack[len(updatedStack)-1])
	}
	return newStacks
}

// Add stacks transformation here
func NewManifest(id string, dependencies []BuildpackMetadataDependency) Manifest {
	var manifestDependencies []ManifestDependency
	for _, dependency := range dependencies {
		manifestDependencies = append(manifestDependencies, ManifestDependency{
			ID:           dependency.ID,
			Name:         dependency.ID,
			Version:      dependency.Version,
			URI:          dependency.URI,
			SHA256:       dependency.SHA256,
			Source:       dependency.Source,
			SourceSHA256: dependency.SourceSHA256,
			Stacks:       UpdateStacks(dependency.Stacks),
		})
	}

	parts := strings.Split(id, ".")

	return Manifest{
		Language: parts[len(parts)-1],
		IncludeFiles: []string{
			"bin/compile",
			"bin/detect",
			"bin/finalize",
			"bin/release",
			"bin/supply",
			"buildpack.toml",
			"manifest.yml",
			"VERSION",
		},
		Dependencies: manifestDependencies,
	}
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
