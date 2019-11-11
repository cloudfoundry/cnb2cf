package untested

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/cloudfoundry/cnb2cf/packager"
)

//go:generate faux -i Installer -o fakes/installer.go
type Installer interface {
	Download(uri, checksum, destination string) error
}

type DependencyPackager struct {
	installer Installer

	scratchDirectory string
	cached           bool
	dev              bool
}

func NewDependencyPackager(scratchDirectory string, cached, dev bool, installer Installer) DependencyPackager {
	return DependencyPackager{
		installer:        installer,
		scratchDirectory: scratchDirectory,
		cached:           cached,
		dev:              dev,
	}
}

func (dp DependencyPackager) Package(dependency cloudnative.BuildpackMetadataDependency, stack string) ([]cloudnative.BuildpackMetadataDependency, error) {
	if !dependency.MatchesStack(stack) {
		return nil, nil
	}

	downloadDir, err := ioutil.TempDir(dp.scratchDirectory, "download")
	if err != nil {
		return nil, err
	}

	buildDir, err := ioutil.TempDir(dp.scratchDirectory, "build")
	if err != nil {
		return nil, err
	}

	var tarFile string
	if dependency.ID == cloudnative.Lifecycle {
		tarFile = filepath.Join(buildDir, dependency.ID+".tgz")
		err := dp.installer.Download(dependency.URI, dependency.SHA256, tarFile)
		if err != nil {
			return nil, fmt.Errorf("failed to download cnb source for %s: %s", dependency.ID, err)
		}
	} else {
		tarFile = filepath.Join(downloadDir, filepath.Base(dependency.Source))
		err := dp.installer.Download(dependency.Source, dependency.SourceSHA256, tarFile)
		if err != nil {
			return nil, fmt.Errorf("failed to download cnb source for %s: %s", dependency.ID, err)
		}
	}

	var dependencies []cloudnative.BuildpackMetadataDependency
	if dependency.ID != cloudnative.Lifecycle {
		if err := packager.ExtractCNBSource(dependency, tarFile, downloadDir); err != nil {
			return nil, fmt.Errorf("failed to extract cnb source for %s: %s", dependency.ID, err)
		}

		tarballPath, sha256, err := packager.BuildCNB(downloadDir, filepath.Join(buildDir, dependency.ID), dp.cached, dependency.Version)
		if err != nil {
			panic(err)
		}

		path, err := packager.FindCNB(downloadDir)
		if err != nil {
			panic(err)
		}

		buildpack, err := cloudnative.ParseBuildpack(filepath.Join(path, "buildpack.toml"))
		if err != nil {
			panic(err)
		}

		if len(buildpack.Orders) > 0 {
			for _, d := range buildpack.Metadata.Dependencies {
				children, err := dp.Package(d, stack)
				if err != nil {
					return nil, err
				}

				dependencies = append(dependencies, children...)
			}
		}
		dependency.URI = fmt.Sprintf("file://%s", tarballPath)
		dependency.SHA256 = sha256
	}

	for i, stack := range dependency.Stacks {
		// Translate stack from org.cloudfoundry.stacks.cflinuxfs3 to just cflinuxfs3
		dependency.Stacks[i] = strings.Split(stack, ".stacks.")[1]
	}

	dependencies = append(dependencies, dependency)

	return dependencies, nil
}
