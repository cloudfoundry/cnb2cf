package node

import (
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"

	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
)

const Dependency = "node"

type Config struct {
	OptimizeMemory bool `yaml:"optimize-memory"`
}

type BuildpackYAML struct {
	Config Config `yaml:"nodejs"`
}

type Contributor struct {
	BuildpackYAML      BuildpackYAML
	buildContribution  bool
	launchContribution bool
	layer              layers.DependencyLayer
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	buildpackYAML, err := LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return Contributor{}, false, err
	}

	plan, wantDependency := context.BuildPlan[Dependency]
	if !wantDependency {
		return Contributor{}, false, nil
	}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return Contributor{}, false, err
	}

	version := plan.Version
	if version == "" {
		if version, err = context.Buildpack.DefaultVersion(Dependency); err != nil {
			return Contributor{}, false, err
		}
	}

	dep, err := deps.Best(Dependency, version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	contributor := Contributor{layer: context.Layers.DependencyLayer(dep), BuildpackYAML: buildpackYAML}

	if _, ok := plan.Metadata["build"]; ok {
		contributor.buildContribution = true
	}

	if _, ok := plan.Metadata["launch"]; ok {
		contributor.launchContribution = true
	}

	return contributor, true, nil
}

func (c Contributor) Contribute() error {
	return c.layer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.SubsequentLine("Expanding to %s", layer.Root)
		if err := helper.ExtractTarGz(artifact, layer.Root, 1); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("NODE_HOME", layer.Root); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("NODE_ENV", "production"); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("NODE_MODULES_CACHE", "true"); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("NODE_VERBOSE", "false"); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("NPM_CONFIG_PRODUCTION", "true"); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("NPM_CONFIG_LOGLEVEL", "error"); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("WEB_MEMORY", "512"); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("WEB_CONCURRENCY", "1"); err != nil {
			return err
		}

		if err := layer.WriteProfile("0_memory_available.sh", memoryAvailable()); err != nil {
			return err
		}

		if c.BuildpackYAML.Config.OptimizeMemory || os.Getenv("OPTIMIZE_MEMORY") == "true" {
			if err := layer.WriteProfile("1_optimize_memory.sh", `export NODE_OPTIONS="--max_old_space_size=$(( $MEMORY_AVAILABLE * 75 / 100 ))"`); err != nil {
				return err
			}
		}

		return nil
	}, c.flags()...)
}

func LoadBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(appRoot, "buildpack.yml")

	if exists, err := helper.FileExists(configFile); err != nil {
		return BuildpackYAML{}, err
	} else if exists {
		file, err := os.Open(configFile)
		if err != nil {
			return BuildpackYAML{}, err
		}
		defer file.Close()

		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return BuildpackYAML{}, err
		}

		err = yaml.Unmarshal(contents, &buildpackYAML)
		if err != nil {
			return BuildpackYAML{}, err
		}
	}
	return buildpackYAML, nil
}

func (c Contributor) flags() []layers.Flag {
	flags := []layers.Flag{layers.Cache}

	if c.buildContribution {
		flags = append(flags, layers.Build)
	}

	if c.launchContribution {
		flags = append(flags, layers.Launch)
	}

	return flags
}

func memoryAvailable() string {
	return `which jq
if [[ $? -eq 0 ]]; then
	MEMORY_AVAILABLE="$(echo $VCAP_APPLICATION | jq .limits.mem)"
fi

if [[ -z "$MEMORY_AVAILABLE" ]]; then
	memory_in_bytes="$(cat /sys/fs/cgroup/memory/memory.limit_in_bytes)"
	MEMORY_AVAILABLE="$(( $memory_in_bytes / ( 1024 * 1024 ) ))"
fi
export MEMORY_AVAILABLE
`
}
