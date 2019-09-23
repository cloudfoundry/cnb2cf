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
	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/cloudfoundry/cnb2cf/packager"
	"github.com/cloudfoundry/libbuildpack"
	cfPackager "github.com/cloudfoundry/libbuildpack/packager"
	"github.com/google/subcommands"
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
	buildpack, err := cloudnative.ParseBuildpack("buildpack.toml")
	if err != nil {
		panic(err)
	}

	// create temporary directory, for what?
	tmpDir, err := ioutil.TempDir("", "cnb2cf")
	if err != nil {
		log.Printf("failed to create temp dir: %s\n", err)
		return subcommands.ExitFailure
	}
	defer os.RemoveAll(tmpDir)

	// create "build" directory inside temp dir
	buildDir := filepath.Join(tmpDir, buildpack.Info.ID, "build")
	err = os.MkdirAll(buildDir, 0777)
	if err != nil {
		panic(err)
	}

	// create and "initialize" manifest.yml, why do we need a manifest.yml?
	var manifest metadata.ManifestYAML
	manifest.Initialize(buildpack.Info.ID)

	// initialize packager
	pkgr := packager.Packager{Dev: p.dev}
	path, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}

	// build the top-level language family CNB
	err = pkgr.BuildCNB(path, filepath.Join(buildDir, buildpack.Info.ID), p.cached, buildpack.Info.Version)
	if err != nil {
		panic(err)
	}

	// update the top-level language family CNB with file path and SHA details
	dependency, err := metadata.UpdateDependency(metadata.Dependency{
		ID:       buildpack.Info.ID,
		Version:  buildpack.Info.Version,
		CFStacks: []string{"org.cloudfoundry.stacks." + p.stack},
	}, filepath.Join(buildDir, buildpack.Info.ID+".tgz"))
	if err != nil {
		panic(err)
	}

	// update manifest with top-level CNB dependency
	manifest.Dependencies = append(manifest.Dependencies, dependency)

	// create the CNB packager
	dependencyPackager := NewDependencyPackager(tmpDir, pkgr, p.cached)

	// package child dependencies of the top-level CNB
	for _, dependency := range buildpack.Metadata.Dependencies {
		dependencies, err := dependencyPackager.Package(dependency)
		if err != nil {
			log.Printf("failed to handle dependency: %s\n", err)
			return subcommands.ExitFailure
		}

		for _, dep := range dependencies {
			manifest.Dependencies = append(manifest.Dependencies, metadata.Dependency{
				Name:         dep.ID,
				ID:           dep.ID,
				Version:      dep.Version,
				URI:          dep.URI,
				SHA256:       dep.SHA256,
				Source:       dep.Source,
				SourceSHA256: dep.SourceSHA256,
				CFStacks:     dep.Stacks,
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

	// TODO: maybe we shouldn't if exists bin dir
	if err := pkgr.WriteBinFromTemplate(dir); err != nil {
		log.Printf("failed to write the shim binaries from the template directory: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	// write out manifest file, TODO: do we still need this?
	if err := manifest.Write(filepath.Join(dir, "manifest.yml")); err != nil {
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

type DependencyPackager struct {
	scratchDirectory string
	packager         packager.Packager
	cached           bool
}

func NewDependencyPackager(scratchDirectory string, packager packager.Packager, cached bool) DependencyPackager {
	return DependencyPackager{
		scratchDirectory: scratchDirectory,
		packager:         packager,
		cached:           cached,
	}
}

func (p DependencyPackager) Package(dependency cloudnative.BuildpackMetadataDependency) ([]cloudnative.BuildpackMetadataDependency, error) {
	depReplaceMe := metadata.Dependency{
		ID:           dependency.ID,
		Version:      dependency.Version,
		URI:          dependency.URI,
		SHA256:       dependency.SHA256,
		Source:       dependency.Source,
		SourceSHA256: dependency.SourceSHA256,
		CFStacks:     dependency.Stacks,
	}

	downloadDir, buildDir, err := makeDirs(filepath.Join(p.scratchDirectory, dependency.ID))
	if err != nil {
		return nil, err
	}

	tarFile := filepath.Join(downloadDir, filepath.Base(dependency.Source))
	fromSource := true
	if dependency.ID == metadata.Lifecycle {
		tarFile = filepath.Join(buildDir, dependency.ID+".tgz")
		fromSource = false
	}

	if err := p.packager.InstallDependency(depReplaceMe, tarFile, fromSource); err != nil {
		return nil, fmt.Errorf("failed to download cnb source for %s: %s", dependency.ID, err)
	}

	var dependencies []cloudnative.BuildpackMetadataDependency
	if dependency.ID != metadata.Lifecycle {
		if err := p.packager.ExtractCNBSource(depReplaceMe, tarFile, downloadDir); err != nil {
			return nil, fmt.Errorf("failed to extract cnb source for %s: %s", dependency.ID, err)
		}

		if err := p.packager.BuildCNB(downloadDir, filepath.Join(buildDir, dependency.ID), p.cached, dependency.Version); err != nil {
			return nil, fmt.Errorf("failed to build cnb from source for %s: %s", dependency.ID, err)
		}

		path, err := p.packager.FindCNB(downloadDir)
		if err != nil {
			panic(err)
		}

		buildpack, err := cloudnative.ParseBuildpack(filepath.Join(path, "buildpack.toml"))
		if err != nil {
			panic(err)
		}

		if len(buildpack.Orders) > 0 {
			for _, d := range buildpack.Metadata.Dependencies {
				children, err := p.Package(d)
				if err != nil {
					return nil, err
				}

				dependencies = append(dependencies, children...)
			}
		}
	}

	depReplaceMe, err = metadata.UpdateDependency(depReplaceMe, filepath.Join(buildDir, dependency.ID+".tgz"))
	if err != nil {
		return nil, fmt.Errorf("failed to update manifest dependency with built cnb for %s: %s", dependency.ID, err)
	}

	dependencies = append(dependencies, cloudnative.BuildpackMetadataDependency{
		ID:           depReplaceMe.ID,
		Version:      depReplaceMe.Version,
		URI:          depReplaceMe.URI,
		SHA256:       depReplaceMe.SHA256,
		Source:       depReplaceMe.Source,
		SourceSHA256: depReplaceMe.SourceSHA256,
		Stacks:       depReplaceMe.CFStacks,
	})

	return dependencies, nil
}

func makeDirs(root string) (string, string, error) {
	downloadDir := filepath.Join(root, "download")
	if err := os.MkdirAll(downloadDir, 0777); err != nil {
		return "", "", err
	}

	buildDir := filepath.Join(root, "build")
	if err := os.MkdirAll(buildDir, 0777); err != nil {
		return "", "", err
	}

	return downloadDir, buildDir, nil
}
