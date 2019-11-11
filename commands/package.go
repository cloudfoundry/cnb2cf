package commands

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/cloudfoundry/cnb2cf/cloudnative/untested"
	"github.com/cloudfoundry/cnb2cf/packager"
	"github.com/cloudfoundry/libbuildpack"
	cfPackager "github.com/cloudfoundry/libbuildpack/packager"
	"github.com/google/subcommands"
	"github.com/rakyll/statik/fs"
)

const PackageUsage = `package -stack <stack> [-cached] [-version <version>] [-cachedir <path to cachedir>] [-manifestpath <optional path to manifest>]:
  when run in a directory that is structured as a shimmed buildpack, creates a zip file.

`

func Fetch(bp cloudnative.Buildpack, installer untested.Installer) error {
	// TODO: fixme
	// package child dependencies of the top-level CNB
	for _, dependency := range bp.Metadata.Dependencies {
		downloadPath := filepath.Join(os.TempDir(), "downloads")
		err := installer.Download(dependency.URI, dependency.SHA256, downloadPath)
		if err != nil {
			return err
		}
	}
	return nil
}

type Package struct {
	cached            bool
	version           string
	cacheDir          string
	stack             string
	buildpackTOMLPath string
	dev               bool
	release           bool
}

func (*Package) Name() string {
	return "package"
}

func (*Package) Synopsis() string {
	return "Create a shimmed buildpack zipfile from the current directory"
}

func (*Package) Usage() string {
	return PackageUsage
}

func (p *Package) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.version, "version", "", "version to package as")
	f.BoolVar(&p.cached, "cached", false, "include dependencies")
	f.StringVar(&p.cacheDir, "cachedir", packager.DefaultCacheDir, "cache dir")
	f.StringVar(&p.stack, "stack", "", "stack to package buildpack for")
	f.BoolVar(&p.dev, "dev", false, "use local dependencies")
	f.BoolVar(&p.release, "release", false, "use released dependencies instead of re-packaging from source")
	f.StringVar(&p.buildpackTOMLPath, "manifestpath", "buildpack.toml", "custom path to a buildpack.toml file")
}

func (p *Package) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// START setup
	tmpDir, err := ioutil.TempDir("", "cnb2cf")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	statikFS, err := fs.New()
	if err != nil {
		panic(err)
	}

	filesystem := cloudnative.NewFilesystem(statikFS)
	dependencyInstaller := cloudnative.NewDependencyInstaller()
	dependencyPackager := untested.NewDependencyPackager(tmpDir, p.cached, p.dev, dependencyInstaller)
	lifecycleHooks := cloudnative.NewLifecycleHooks(filesystem)
	// END setup

	// Parse current buildpack.toml
	buildpack, err := cloudnative.ParseBuildpack("buildpack.toml")
	if err != nil {
		panic(err)
	}

	if p.version == "" {
		fmt.Printf("--version is a required flag")
		os.Exit(1)
	}

	if p.stack == "" {
		fmt.Printf("--stack is a required flag")
		os.Exit(1)
	}

	buildpack.Info.Version = p.version

	// create "build" directory inside temp dir
	buildDir := filepath.Join(tmpDir, buildpack.Info.ID, "build")
	err = os.MkdirAll(buildDir, 0777)
	if err != nil {
		panic(err)
	}

	// package child dependencies of the top-level CNB
	var dependencies []cloudnative.BuildpackMetadataDependency
	if p.release {
		// TODO: fix the if branch
		err := Fetch(buildpack, dependencyInstaller)
		if err != nil {
			panic(err)
		}
		dependencies = buildpack.Metadata.Dependencies
	} else {

		for _, dependency := range buildpack.Metadata.Dependencies {
			var deps []cloudnative.BuildpackMetadataDependency
			var err error
			deps, err = dependencyPackager.Package(dependency, p.stack)
			if err != nil {
				log.Printf("failed to handle dependency: %s\n", err)
				return subcommands.ExitFailure
			}

			for _, dep := range deps {
				dependencies = append(dependencies, cloudnative.BuildpackMetadataDependency{
					ID:           dep.ID,
					Version:      dep.Version,
					URI:          dep.URI,
					SHA256:       dep.SHA256,
					Source:       dep.Source,
					SourceSHA256: dep.SourceSHA256,
					Stacks:       dep.Stacks,
				})
			}
		}
	}

	dir, err := ioutil.TempDir("", "buildpack-packager")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	// write the buildpack.toml to disk
	bpTOMLFile, err := os.OpenFile(filepath.Join(dir, "buildpack.toml"), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	err = toml.NewEncoder(bpTOMLFile).Encode(buildpack)
	if err != nil {
		panic(err)
	}

	for _, hook := range []string{"compile", "detect", "finalize", "release", "supply"} {
		if err := lifecycleHooks.Install(hook, dir); err != nil {
			panic(err)
		}
	}

	manifest := cloudnative.NewManifest(buildpack.Info.ID, dependencies)
	if err := cloudnative.WriteManifest(manifest, filepath.Join(dir, "manifest.yml")); err != nil {
		log.Printf("failed to update manifest: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	// Uses V2B Packager to ensure cached dependencies are set up correctly
	// Cached is always true, because the CNBs are being cached (even if their internal dependencies aren't) within the shimmed buildpack
	zipFile, err := cfPackager.Package(dir, p.cacheDir, buildpack.Info.Version, p.stack, true)
	if err != nil {
		return subcommands.ExitFailure
	}

	newName := filepath.Base(zipFile)
	if !p.cached {
		newName = strings.Replace(newName, "-cached", "", 1)
	}

	if err := libbuildpack.CopyFile(zipFile, newName); err != nil {
		log.Print(err.Error())
		return subcommands.ExitFailure
	}

	log.Printf("Packaged Shimmed Buildpack at: %s", filepath.Base(newName))

	return subcommands.ExitSuccess
}
