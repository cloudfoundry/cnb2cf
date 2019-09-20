package shims

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/cloudfoundry/libbuildpack"
)

type stack struct {
	ID string `toml:"id"`
}

type CNBInstaller struct {
	*libbuildpack.Installer
	manifest *libbuildpack.Manifest
}

func NewCNBInstaller(manifest *libbuildpack.Manifest) *CNBInstaller {
	return &CNBInstaller{libbuildpack.NewInstaller(manifest), manifest}
}

func (c *CNBInstaller) InstallCNBs(orderFile string, installDir string) error {
	buildpack, err := ParseBuildpackTOML(orderFile)
	if err != nil {
		return err
	}

	bpSet := map[string]interface{}{
		buildpack.Info.ID: nil,
	}
	for _, order := range buildpack.Order {
		for _, bp := range order.Groups {
			bpSet[bp.ID] = nil
		}
	}

	for buildpack := range bpSet {
		versions := c.manifest.AllDependencyVersions(buildpack)
		if len(versions) != 1 {
			return fmt.Errorf("unable to find a unique version of %s in the manifest", buildpack)
		}

		buildpackDest := filepath.Join(installDir, buildpack, versions[0])
		if exists, err := libbuildpack.FileExists(buildpackDest); err != nil {
			return err
		} else if exists {
			continue
		}

		err := c.InstallOnlyVersion(buildpack, buildpackDest)
		if err != nil {
			return err
		}

		err = c.InstallCNBs(filepath.Join(buildpackDest, "buildpack.toml"), installDir)
		if err != nil {
			panic(err)
		}

		err = os.Symlink(buildpackDest, filepath.Join(installDir, buildpack, "latest"))
		if err != nil {
			return err
		}
	}

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

	if err := c.InstallOnlyVersion(V3LifecycleDep, tempDir); err != nil {
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
