package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"

	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/cloudfoundry/cnb2cf/packager"
	cfPackager "github.com/cloudfoundry/libbuildpack/packager"
	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(&packageCmd{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}

type packageCmd struct {
	cached   bool
	version  string
	cacheDir string
	stack    string
	dev      bool
}

func (*packageCmd) Name() string { return "package" }
func (*packageCmd) Synopsis() string {
	return "Create a shimmed buildpack zipfile from the current directory"
}
func (*packageCmd) Usage() string {
	return `package -stack <stack> [-cached] [-version <version>] [-cachedir <path to cachedir>]:
  When run in a directory that is structured as a shimmed buildpack, creates a zip file.

`
}
func (p *packageCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.version, "version", "", "version to package as")
	f.BoolVar(&p.cached, "cached", false, "include dependencies")
	f.StringVar(&p.cacheDir, "cachedir", packager.DefaultCacheDir, "cache dir")
	f.StringVar(&p.stack, "stack", "", "stack to package buildpack for")
	f.BoolVar(&p.dev, "dev", false, "use local dependencies")
}

func (p *packageCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var manifest metadata.ManifestYAML
	var bpTOML metadata.BuildpackToml

	if err := bpTOML.Load("buildpack.toml"); err != nil {
		log.Printf("failed to load buildpack.toml: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	tmpDir, err := ioutil.TempDir("", "cnb2cf")
	if err != nil {
		log.Printf("failed to create temp dir: %s\n", err.Error())
		return subcommands.ExitFailure
	}
	defer os.RemoveAll(tmpDir)

	manifest.Initialize(bpTOML.Info.ID)
	pkgr := packager.Packager{Dev: p.dev}

	for _, d := range bpTOML.Metadata.Dependencies {
		downloadDir, buildDir, err := makeDirs(filepath.Join(tmpDir, d.ID))
		if err != nil {
			log.Println(err.Error())
			return subcommands.ExitFailure
		}

		tarFile := filepath.Join(downloadDir, filepath.Base(d.Source))
		fromSource := true
		if d.ID == metadata.Lifecycle {
			tarFile = filepath.Join(buildDir, d.ID+".tgz")
			fromSource = false
		}

		if err := pkgr.InstallDependency(d, tarFile, fromSource); err != nil {
			log.Printf("failed to download CNB source for %s: %s\n", d.ID, err.Error())
			return subcommands.ExitFailure
		}

		if d.ID != metadata.Lifecycle {
			if err := pkgr.ExtractCNBSource(d, tarFile, downloadDir); err != nil {
				log.Printf("failed to extract CNB source for %s: %s\n", d.ID, err.Error())
				return subcommands.ExitFailure
			}

			if err := pkgr.BuildCNB(downloadDir, filepath.Join(buildDir, d.ID), p.cached, d.Version); err != nil {
				log.Printf("failed to build CNB from source for %s: %s\n", d.ID, err.Error())
				return subcommands.ExitFailure
			}
		}

		if err := d.UpdateDependency(filepath.Join(buildDir, d.ID+".tgz")); err != nil {
			log.Printf("failed to update manifest dependency with built CNB for %s: %s\n", d.ID, err.Error())
			return subcommands.ExitFailure
		}

		manifest.Dependencies = append(manifest.Dependencies, d)
	}

	// Copies current directory into tempdir for packaging
	dir, err := cfPackager.CopyDirectory(".")
	if err != nil {
		log.Printf("failed to copy buildpack dir: %s\n", err.Error())
		return subcommands.ExitFailure
	}
	defer os.RemoveAll(dir)

	log.Printf("Manifest: %v", manifest)
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
