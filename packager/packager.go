package packager

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cloudfoundry/cnb2cf/metadata"
	"github.com/cloudfoundry/cnb2cf/utils"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libcfbuildpack/packager/cnbpackager"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var DefaultCacheDir = filepath.Join(os.Getenv("HOME"), ".cnb2cf", "cache")

func InstallCNBSource(dep metadata.V2Dependency, dstFile string) error {
	if err := downloadFile(dep.Source, dstFile); err != nil {
		return err
	}

	return checkSHA256(dstFile, dep.SourceSHA256)
}

func ExtractCNBSource(dep metadata.V2Dependency, srcFile, outputDir string) error {
	if strings.HasSuffix(dep.Name, ".zip") {
		return libbuildpack.ExtractZip(srcFile, outputDir)
	}

	if strings.HasSuffix(dep.Name, ".tar.xz") {
		return libbuildpack.ExtractTarXz(srcFile, outputDir)
	}

	return libbuildpack.ExtractTarGz(srcFile, outputDir)
}

func BuildCNB(extractDir, outputDir string, cached bool) error {
	foundSrc, err := FindCNB(extractDir)
	if err != nil {
		return err
	}

	packager, err := cnbpackager.New(foundSrc, outputDir)
	if err != nil {
		return err
	}

	if err := packager.Create(cached); err != nil {
		return err
	}
	return packager.Archive(cached)
}

// FindCNB returns the path to the cnb source if it can find a single buildpack.toml
// in the top level dir or within one directory
// This is to support source tar files with a root directory (github release structure)
func FindCNB(extractDir string) (string, error) {
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

func UpdateDependency(dep *metadata.V2Dependency, depPath string) error {
	dep.URI = fmt.Sprintf("file://%s", depPath)
	sha, err := getSHA256(depPath)
	if err != nil {
		return err
	}

	dep.SHA256 = hex.EncodeToString(sha[:])
	return nil
}

func downloadFile(url, destFile string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("could not download: %d", resp.StatusCode)
	}

	return utils.WriteToFile(resp.Body, destFile, 0666)
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
