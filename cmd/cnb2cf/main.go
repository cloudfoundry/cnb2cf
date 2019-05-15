package main

//go:generate statik -src=./template

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"

	"github.com/cloudfoundry/cnb2cf/creator"
	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/cloudfoundry/cnb2cf/packager"
	cfPackager "github.com/cloudfoundry/libbuildpack/packager"
	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(&createCmd{}, "")
	subcommands.Register(&packageCmd{}, "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}

type createCmd struct {
	config string
}

func (*createCmd) Name() string     { return "create" }
func (*createCmd) Synopsis() string { return "Create a shimmed buildpack from a config file" }
func (*createCmd) Usage() string {
	return `create -config <path to config file>:
  Generates a shimmed buildpack zip file from the configuration.
`
}

func (c *createCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.config, "config", "", "Path to config file")
}
func (c *createCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	config, err := creator.LoadConfig(c.config)
	if err != nil {
		log.Printf("failed to load config: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Printf("failed to create tempdir: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	defer os.RemoveAll(tempDir)

	outputDir := "."

	if err := creator.CreateBuildpack(config, tempDir); err != nil {
		log.Printf("failed to convert buildpack to shim: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	if err := creator.CreateZip(config, tempDir, outputDir); err != nil {
		log.Printf("failed to create zip: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
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
	if err := manifest.Load("manifest.yml"); err != nil {
		log.Printf("failed to load manifest.yml: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	tmpDir, err := ioutil.TempDir("", "cnb2cf")
	if err != nil {
		log.Printf("failed to create temp dir: %s\n", err.Error())
		return subcommands.ExitFailure
	}
	defer os.RemoveAll(tmpDir)

	pkgr := packager.Packager{Dev: p.dev}

	for i, d := range manifest.Dependencies {
		if d.Name == "lifecycle" {
			continue
		}

		downloadDir, buildDir, err := makeDirs(filepath.Join(tmpDir, d.Name))
		if err != nil {
			log.Println(err.Error())
			return subcommands.ExitFailure
		}

		tarFile := filepath.Join(downloadDir, filepath.Base(d.Source))

		if err := pkgr.InstallCNBSource(d, tarFile); err != nil {
			log.Printf("failed to download CNB source for %s: %s\n", d.Name, err.Error())
			return subcommands.ExitFailure
		}

		if err := pkgr.ExtractCNBSource(d, tarFile, downloadDir); err != nil {
			log.Printf("failed to extract CNB source for %s: %s\n", d.Name, err.Error())
			return subcommands.ExitFailure
		}

		if err := pkgr.BuildCNB(downloadDir, filepath.Join(buildDir, d.Name), p.cached); err != nil {
			log.Printf("failed to build CNB from source for %s: %s\n", d.Name, err.Error())
			return subcommands.ExitFailure
		}

		currentDepName := d.Name
		if p.cached {
			currentDepName += "-cached"
		}

		if err := pkgr.UpdateDependency(&d, filepath.Join(buildDir, currentDepName+".tgz")); err != nil {
			log.Printf("failed to update manifest dependency with built CNB for %s: %s\n", d.Name, err.Error())
			return subcommands.ExitFailure
		}

		manifest.Dependencies[i] = d
	}

	// Copies current directory into tempdir for packaging
	dir, err := cfPackager.CopyDirectory(".")
	if err != nil {
		log.Printf("failed to copy buildpack dir: %s\n", err.Error())
		return subcommands.ExitFailure
	}
	defer os.RemoveAll(dir)

	if err := manifest.Write(filepath.Join(dir, "manifest.yml")); err != nil {
		log.Printf("failed to update manifest: %s\n", err.Error())
		return subcommands.ExitFailure
	}

	if p.version == "" {
		v, err := ioutil.ReadFile("VERSION")
		if err != nil {
			log.Printf("-version was not set and failed to read VERSION file: %s\n", err.Error())
			return subcommands.ExitFailure
		}
		p.version = strings.TrimSpace(string(v))
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
