package shimmer

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Language   string         `yaml:"language"`
	Version    string         `yaml:"version"`
	Stack      string         `yaml:"stack"`
	Buildpacks []V2Dependency `yaml:"buildpacks"`
	Groups     []CNBGroup     `yaml:"groups"`
}

func LoadConfig(path string) (Config, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	config := Config{}
	if err := yaml.Unmarshal(contents, &config); err != nil {
		return Config{}, err
	}

	if err := ValidateConfig(config); err != nil {
		return Config{}, errors.Wrapf(err, "%s config error", path)
	}

	return config, nil
}

func ValidateConfig(cfg Config) error {
	if len(cfg.Groups) == 0 {
		return errors.New("groups must not be empty")
	}

	if len(cfg.Buildpacks) == 0 {
		return errors.New("buildpacks must not be empty")
	}

	if cfg.Language == "" {
		return errors.New("language must not be empty")
	}

	if cfg.Version == "" {
		return errors.New("version must not be empty")
	}

	if cfg.Stack == "" {
		return errors.New("stack must not be empty")
	}

	return validateBuildpackIDs(cfg)
}

func validateBuildpackIDs(cfg Config) error {
	buildpackIDs := []string{}
	for _, bp := range cfg.Buildpacks {
		buildpackIDs = append(buildpackIDs, bp.Name)
	}

	groupIDs := map[string]bool{}
	for _, group := range cfg.Groups {
		for _, bp := range group.Buildpacks {
			groupIDs[bp.ID] = true
		}
	}

	if len(groupIDs) != len(buildpackIDs) {
		return fmt.Errorf("buildpack names and group ids do not match")
	}

	for _, id := range buildpackIDs {
		if _, ok := groupIDs[id]; !ok {
			return fmt.Errorf("buildpack name %s does not exist in any groups", id)
		}
	}
	return nil
}
