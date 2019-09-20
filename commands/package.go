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

	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/cloudfoundry/cnb2cf/packager"
	"github.com/cloudfoundry/libbuildpack"
	cfPackager "github.com/cloudfoundry/libbuildpack/packager"
	"github.com/google/subcommands"
)

type Package struct {
	cached   bool
	version  string
	cacheDir string
	stack    string
	dev      bool
}

func (*Package) Name() string { return "package" }
func (*Package) Synopsis() string {
	return "Create a shimmed buildpack zipfile from the current directory"
}
func (*Package) Usage() string {
	return `package -stack <stack> [-cached] [-version <version>] [-cachedir <path to cachedir>]:
  When run in a directory that is structured as a shimmed buildpack, creates a zip file.

`
}
func (p *Package) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.version, "version", "", "version to package as")
	f.BoolVar(&p.cached, "cached", false, "include dependencies")
	f.StringVar(&p.cacheDir, "cachedir", packager.DefaultCacheDir, "cache dir")
	f.StringVar(&p.stack, "stack", "", "stack to package buildpack for")
	f.BoolVar(&p.dev, "dev", false, "use local dependencies")
}

func (p *Package) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var bpTOML metadata.BuildpackToml
	if err := bpTOML.Load("buildpack.toml"); err != nil {
		log.Printf("failed to load buildpack.toml: %s\n", err)
		return subcommands.ExitFailure
	}

	tmpDir, err := ioutil.TempDir("", "cnb2cf")
	if err != nil {
		log.Printf("failed to create temp dir: %s\n", err)
		return subcommands.ExitFailure
	}
	defer os.RemoveAll(tmpDir)

	buildDir := filepath.Join(tmpDir, bpTOML.Info.ID, "build")
	err = os.MkdirAll(buildDir, 0777)
	if err != nil {
		panic(err)
	}

	var manifest metadata.ManifestYAML
	manifest.Initialize(bpTOML.Info.ID)

	pkgr := packager.Packager{Dev: p.dev}
	path, err := filepath.Abs(".")
	if err != nil {
		panic(err)
	}

	err = pkgr.BuildCNB(path, filepath.Join(buildDir, bpTOML.Info.ID), p.cached, bpTOML.Info.Version)
	if err != nil {
		panic(err)
	}

	dependency, err := metadata.UpdateDependency(metadata.Dependency{
		ID:       bpTOML.Info.ID,
		Version:  bpTOML.Info.Version,
		CFStacks: []string{"org.cloudfoundry.stacks." + p.stack},
	}, filepath.Join(buildDir, bpTOML.Info.ID+".tgz"))
	if err != nil {
		panic(err)
	}

	manifest.Dependencies = append(manifest.Dependencies, dependency)

	cnbPackager := CNBPackager{
		scratchDirectory: tmpDir,
		packager:         pkgr,
		cached:           p.cached,
	}

	for _, d := range bpTOML.Metadata.Dependencies {
		dependencies, err := cnbPackager.Package(d)
		if err != nil {
			log.Printf("failed to handle dependency: %s\n", err)
			return subcommands.ExitFailure
		}

		manifest.Dependencies = append(manifest.Dependencies, dependencies...)
	}

	// Copies current directory into tempdir for packaging
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

	if err := manifest.Write(filepath.Join(dir, "manifest.yml")); err != nil {
		log.Printf("failed to update manifest: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	if p.version == "" {
		p.version = bpTOML.Info.Version
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

type CNBPackager struct {
	scratchDirectory string
	packager         packager.Packager
	cached           bool
}

func (c CNBPackager) Package(dependency metadata.Dependency) ([]metadata.Dependency, error) {
	downloadDir, buildDir, err := makeDirs(filepath.Join(c.scratchDirectory, dependency.ID))
	if err != nil {
		return nil, err
	}

	tarFile := filepath.Join(downloadDir, filepath.Base(dependency.Source))
	fromSource := true
	if dependency.ID == metadata.Lifecycle {
		tarFile = filepath.Join(buildDir, dependency.ID+".tgz")
		fromSource = false
	}

	if err := c.packager.InstallDependency(dependency, tarFile, fromSource); err != nil {
		return nil, fmt.Errorf("failed to download cnb source for %s: %s", dependency.ID, err)
	}

	var dependencies []metadata.Dependency
	if dependency.ID != metadata.Lifecycle {
		if err := c.packager.ExtractCNBSource(dependency, tarFile, downloadDir); err != nil {
			return nil, fmt.Errorf("failed to extract cnb source for %s: %s", dependency.ID, err)
		}

		if err := c.packager.BuildCNB(downloadDir, filepath.Join(buildDir, dependency.ID), c.cached, dependency.Version); err != nil {
			return nil, fmt.Errorf("failed to build cnb from source for %s: %s", dependency.ID, err)
		}

		path, err := c.packager.FindCNB(downloadDir)
		if err != nil {
			panic(err)
		}

		var bpTOML metadata.BuildpackToml
		if err := bpTOML.Load(filepath.Join(path, "buildpack.toml")); err != nil {
			return nil, fmt.Errorf("failed to load %s: %s", filepath.Join(path, "buildpack.toml"), err)
		}

		if len(bpTOML.Order) > 0 {
			for _, d := range bpTOML.Metadata.Dependencies {
				children, err := c.Package(d)
				if err != nil {
					return nil, err
				}
				dependencies = append(dependencies, children...)
			}
		}
	}

	dependency, err = metadata.UpdateDependency(dependency, filepath.Join(buildDir, dependency.ID+".tgz"))
	if err != nil {
		return nil, fmt.Errorf("failed to update manifest dependency with built cnb for %s: %s", dependency.ID, err)
	}

	dependencies = append(dependencies, dependency)

	return dependencies, nil
}
