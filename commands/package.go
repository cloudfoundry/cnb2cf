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

	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/cloudfoundry/cnb2cf/cloudnative/untested"
	"github.com/cloudfoundry/cnb2cf/packager"
	"github.com/cloudfoundry/libbuildpack"
	cfPackager "github.com/cloudfoundry/libbuildpack/packager"
	"github.com/google/subcommands"
	"github.com/rakyll/statik/fs"
)

const PackageUsage = `package -stack <stack> [-cached] [-version <version>] [-cachedir <path to cachedir>]:
  When run in a directory that is structured as a shimmed buildpack, creates a zip file.

`

type Package struct {
	cached   bool
	version  string
	cacheDir string
	stack    string
	dev      bool
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

	buildpack, err := cloudnative.ParseBuildpack("buildpack.toml")
	if err != nil {
		panic(err)
	}

	// create "build" directory inside temp dir
	buildDir := filepath.Join(tmpDir, buildpack.Info.ID, "build")
	err = os.MkdirAll(buildDir, 0777)
	if err != nil {
		panic(err)
	}

	path, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}

	// build the top-level language family CNB
	tarballPath, sha256, err := packager.BuildCNB(path, filepath.Join(buildDir, buildpack.Info.ID), p.cached, buildpack.Info.Version)
	if err != nil {
		panic(err)
	}

	var dependencies []cloudnative.BuildpackMetadataDependency

	dependencies = append(dependencies, cloudnative.BuildpackMetadataDependency{
		ID:      buildpack.Info.ID,
		Version: buildpack.Info.Version,
		URI:     fmt.Sprintf("file://%s", tarballPath),
		SHA256:  sha256,
		Stacks:  []string{p.stack},
	})

	// package child dependencies of the top-level CNB
	for _, dependency := range buildpack.Metadata.Dependencies {
		deps, err := dependencyPackager.Package(dependency)
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

	// Copies current directory into tempdir for packaging, TODO: understand what is being copied here and confirm this still needs to be done
	dir, err := cfPackager.CopyDirectory(".")
	if err != nil {
		log.Printf("failed to copy buildpack dir: %s\n", err.Error())
		return subcommands.ExitFailure
	}
	defer os.RemoveAll(dir)

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

	if p.version == "" {
		p.version = buildpack.Info.Version
	}

	// Uses V2B Packager to ensure cached dependencies are set up correctly
	// Cached is always true, because the CNBs are being cached (even if their internal dependencies aren't) within the shimmed buildpack
	zipFile, err := cfPackager.Package(dir, p.cacheDir, p.version, p.stack, true)
	if err != nil {
		log.Printf("failed to package CF buildpack: %s\n", err.Error())
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
