package shims

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libcfbuildpack/helper"
)

type stack struct {
	ID string `toml:"id"`
}

//go:generate faux --interface DepInstaller --output fakes/depinstaller.go
type DepInstaller interface {
	InstallDependency(dep libbuildpack.Dependency, outputDir string) error
	InstallOnlyVersion(depName string, installDir string) error
}

type CNBInstaller struct {
	depInstaller DepInstaller
	manifest     *libbuildpack.Manifest
}

func NewCNBInstaller(manifest *libbuildpack.Manifest, depInstaller DepInstaller) *CNBInstaller {
	return &CNBInstaller{depInstaller: depInstaller, manifest: manifest}
}

// Really feels like there are two things that are happening here
// 1) we are downloading a bunch of shit to specific locations
//
// 2) we symlink it all to latest

func (c *CNBInstaller) InstallCNBs(orderFile string, installDir string) error {
	buildpack, err := ParseBuildpackTOML(orderFile)
	if err != nil {
		return err
	}

	paths, err := c.DownloadCNBs(buildpack, installDir)
	if err != nil {
		return err
	}
	for _, path := range paths {
		dir := filepath.Dir(path)
		err = os.Symlink(path, filepath.Join(dir, "latest"))
		if err != nil {
			return err
		}
	}

	return nil
}

// install all cnbs and return strings to their paths
func (c *CNBInstaller) DownloadCNBs(buildpack BuildpackTOML, installDir string) ([]string, error) {
	var result []string

	bpSet := map[string]interface{}{}

	for _, order := range buildpack.Order {
		for _, bp := range order.Groups {
			bpSet[bp.ID] = nil
		}
	}

	for buildpack := range bpSet {
		versions := c.manifest.AllDependencyVersions(buildpack)
		if len(versions) != 1 {
			return []string{}, fmt.Errorf("unable to find a unique version of %s in the manifest", buildpack)
		}

		buildpackDest := filepath.Join(installDir, buildpack, versions[0])
		if exists, err := libbuildpack.FileExists(buildpackDest); err != nil {
			return []string{}, err
		} else if exists {
			continue
		}

		err := c.depInstaller.InstallOnlyVersion(buildpack, buildpackDest)
		if err != nil {
			return []string{}, err
		}

		result = append(result, buildpackDest)

		// TODO: this code below should be deprecated once we no longer need to recursivly shim
		nextBPTOML := filepath.Join(buildpackDest, "buildpack.toml")
		exists, err := helper.FileExists(nextBPTOML)
		if err != nil {
			return []string{}, err
		}

		if exists {
			nextBuildpack, err := ParseBuildpackTOML(nextBPTOML)
			if err != nil {
				return []string{}, err
			}
			nextPaths, err := c.DownloadCNBs(nextBuildpack, installDir)
			if err != nil {
				return []string{}, fmt.Errorf("error installing sub-cnb: %s", err.Error())
			}
			result = append(result, nextPaths...)
		}
	}

	return result, nil
}

func (c *CNBInstaller) SymlinkToLatest(paths []string, installDir string) error {
	return nil
}

func (c *CNBInstaller) FindCNB(extractDir string) (string, error) {
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

func (c *CNBInstaller) InstallLifecycle(dst string) error {
	tempDir, err := ioutil.TempDir("", "lifecycle")
	if err != nil {
		return errors.Wrap(err, "InstallLifecycle issue creating tempdir")
	}

	defer os.RemoveAll(tempDir)

	if err := c.depInstaller.InstallOnlyVersion(V3LifecycleDep, tempDir); err != nil {
		return err
	}

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	firstDir, err := filepath.Glob(filepath.Join(tempDir, "*"))
	if err != nil {
		return err
	}

	if len(firstDir) != 2 {
		return errors.Errorf("issue unpacking lifecycle : incorrect dir format : %s", firstDir)
	}

	for _, binary := range []string{V3Detector, V3Builder, V3Launcher} {
		srcBinary := filepath.Join(firstDir[0], binary)
		dstBinary := filepath.Join(dst, binary)
		if err := os.Rename(srcBinary, dstBinary); err != nil {
			return errors.Wrapf(err, "issue copying lifecycle binary: %s", srcBinary)
		}
	}

	return nil
}
