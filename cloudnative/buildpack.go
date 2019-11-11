package cloudnative

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

type Buildpack struct {
	API      string            `toml:"api"`
	Info     BuildpackInfo     `toml:"buildpack"`
	Metadata BuildpackMetadata `toml:"metadata"`
	Orders   []BuildpackOrder  `toml:"order"`
}

type BuildpackInfo struct {
	ID      string `toml:"id"`
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type BuildpackMetadata struct {
	IncludeFiles []string                      `toml:"include_files"`
	Dependencies []BuildpackMetadataDependency `toml:"dependencies"`
}

type BuildpackOrder struct {
	Groups []BuildpackOrderGroup `toml:"group"`
}

type BuildpackOrderGroup struct {
	ID      string `toml:"id"`
	Version string `toml:"version"`
}

func ParseBuildpack(path string) (Buildpack, error) {
	var buildpack Buildpack
	_, err := toml.DecodeFile(path, &buildpack)
	if err != nil {
		return Buildpack{}, fmt.Errorf("failed to parse %s: %s", path, err)
	}

	return buildpack, nil
}

type BuildpackMetadataDependency struct {
	ID      string `toml:"id"`
	Version string `toml:"version"`

	URI    string `toml:"uri"`
	SHA256 string `toml:"sha256"`

	Source       string `toml:"source"`
	SourceSHA256 string `toml:"source_sha256"`

	Stacks []string `toml:"stacks"`
}

func (bpDep BuildpackMetadataDependency) MatchesStack(stackName string) bool {
	for _, stack := range bpDep.Stacks {
		if strings.HasSuffix(stack, stackName) {
			return true
		}
	}
	return false
}
