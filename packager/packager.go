package packager

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
)

var DefaultCacheDir = filepath.Join(os.Getenv("HOME"), ".cnb2cf", "cache")

type Packager struct {
	Dev bool
}

func (p *Packager) InstallCNBSource(dep metadata.V2Dependency, dest string) error {
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

	if err := packager.DownloadFromURI(dep.Source, dest); err != nil {
		return err
	}

	return libbuildpack.CheckSha256(dest, dep.SourceSHA256)
}

func (p *Packager) ExtractCNBSource(dep metadata.V2Dependency, src, dstDir string) error {
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

func (p *Packager) UpdateDependency(dep *metadata.V2Dependency, depPath string) error {
	dep.URI = fmt.Sprintf("file://%s", depPath)
	sha, err := getSHA256(depPath)
	if err != nil {
		return err
	}

	dep.SHA256 = hex.EncodeToString(sha[:])
	return nil
}

func checkSHA256(filePath, expectedSha256 string) error {
	sum, err := getSHA256(filePath)
	if err != nil {
		return err
	}

	actualSha256 := hex.EncodeToString(sum[:])

	if actualSha256 != expectedSha256 {
		return fmt.Errorf("dependency sha256 mismatch: expected sha256 %s, actual sha256 %s", expectedSha256, actualSha256)
	}
	return nil
}

func getSHA256(path string) ([32]byte, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}

	return sha256.Sum256(content), nil
}
