package packager

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/packager"
	"github.com/cloudfoundry/libcfbuildpack/packager/cnbpackager"
	"github.com/pkg/errors"
	"github.com/rakyll/statik/fs"

	_ "github.com/cloudfoundry/cnb2cf/statik"
)

var DefaultCacheDir = filepath.Join(os.Getenv("HOME"), ".cnb2cf", "cache")

type Packager struct {
	Dev bool
}

func (p *Packager) InstallDependency(dep metadata.Dependency, dest string, source bool) error {
	if p.Dev {
		info, err := os.Stat(dep.Source)
		exists := !os.IsNotExist(err)
		if exists && err != nil {
			return err
		}

		if exists && info.IsDir() {
			if err := os.MkdirAll(dest, 0755); err != nil {
				return err
			}
			return libbuildpack.CopyDirectory(dep.Source, dest)
		}
	}

	uri := dep.Source
	sha := dep.SourceSHA256
	if !source {
		uri = dep.URI
		sha = dep.SHA256
	}

	if err := packager.DownloadFromURI(uri, dest); err != nil {
		return err
	}

	return libbuildpack.CheckSha256(dest, sha)
}

func (p *Packager) ExtractCNBSource(dep metadata.Dependency, src, dstDir string) error {
	if strings.HasSuffix(dep.Source, "/") {
		return libbuildpack.CopyDirectory(src, dstDir)
	}

	if strings.HasSuffix(dep.Source, ".zip") {
		return libbuildpack.ExtractZip(src, dstDir)
	}

	if strings.HasSuffix(dep.Source, ".tar.xz") {
		return libbuildpack.ExtractTarXz(src, dstDir)
	}

	return libbuildpack.ExtractTarGz(src, dstDir)
}

func (p *Packager) BuildCNB(extractDir, outputDir string, cached bool, version string) error {
	foundSrc, err := p.FindCNB(extractDir)
	if err != nil {
		return err
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	globalCacheDir := filepath.Join(usr.HomeDir, cnbpackager.DefaultCacheBase)

	packager, err := cnbpackager.New(foundSrc, outputDir, version, globalCacheDir)
	if err != nil {
		return err
	}

	if err := packager.Create(cached); err != nil {
		return err
	}
	return packager.Archive()
}

// FindCNB returns the path to the cnb source if it can find a single buildpack.toml
// in the top level dir or within one directory
// This is to support source tar files with a root directory (github release structure)
func (p *Packager) FindCNB(extractDir string) (string, error) {
	buildpackTOML := filepath.Join(extractDir, "buildpack.toml")
	if _, err := os.Stat(buildpackTOML); err == nil {
		return filepath.Dir(buildpackTOML), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	paths, err := filepath.Glob(filepath.Join(extractDir, "*", "buildpack.toml"))
	if err != nil {
		return "", err
	}

	if len(paths) < 1 {
		return "", errors.New("failed to find find cnb source: no buildpack.toml")
	}

	if len(paths) > 1 {
		return "", errors.New("failed to find find cnb source: found multiple buildpack.toml files")
	}

	return filepath.Dir(paths[0]), nil
}

func (p *Packager) WriteBinFromTemplate(dir string) error {
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, os.ModePerm); err != nil {
		return errors.Wrap(err, "failed to make bin directory")
	}

	statikFS, err := fs.New()
	if err != nil {
		return err
	}

	binFiles := []string{
		"compile",
		"detect",
		"finalize",
		"release",
		"supply",
	}

	for _, file := range binFiles {
		output, err := fs.ReadFile(statikFS, fmt.Sprintf("/bin/%s", file))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to read /bin/%s", file))
		}
		if err := ioutil.WriteFile(filepath.Join(binDir, file), output, os.ModePerm); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to write /bin/%s", file))
		}
	}

	return nil
}
