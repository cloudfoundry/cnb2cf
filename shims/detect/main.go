package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/cnb2cf/cloudnative"
	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/pkg/errors"
)

func main() {
	var logger = libbuildpack.NewLogger(ioutil.Discard)
	if len(os.Args) != 2 {
		logger.Error("Incorrect number of arguments")
		os.Exit(1)
	}

	if err := detect(logger); err != nil {
		logger.Error("Failed detect step: %s", err)
		os.Exit(1)
	}
}

func detect(logger *libbuildpack.Logger) error {
	v2AppDir := os.Args[1]

	tempDir, err := ioutil.TempDir("", "temp")
	if err != nil {
		return errors.Wrap(err, "unable to create temp dir")
	}
	defer os.RemoveAll(tempDir)

	if err := os.MkdirAll(shims.V3BuildpacksDir, 0777); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(shims.V3PlatformDir, "env"), os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(shims.V3MetadataDir, 0777); err != nil {
		return err
	}

	v2BuildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		return err
	}

	manifest, err := libbuildpack.NewManifest(v2BuildpackDir, logger, time.Now())
	if err != nil {
		return err
	}

	detectExecPath := filepath.Join(tempDir, shims.V3Detector)
	executable := pexec.NewExecutable(detectExecPath, lager.NewLogger("detect"))

	detector := shims.Detector{
		V3LifecycleDir:  tempDir,
		AppDir:          v2AppDir,
		V3BuildpacksDir: shims.V3BuildpacksDir,
		V3PlatformDir:   shims.V3PlatformDir,
		OrderMetadata:   filepath.Join(v2BuildpackDir, "buildpack.toml"),
		GroupMetadata:   filepath.Join(shims.V3MetadataDir, "group.toml"),
		PlanMetadata:    filepath.Join(shims.V3MetadataDir, "plan.toml"),
		Installer:       shims.NewCNBInstaller(manifest, libbuildpack.NewInstaller(manifest)),
		Environment:     cloudnative.NewEnvironment(),
		Executor:        executable,
	}

	return detector.Detect()
}
