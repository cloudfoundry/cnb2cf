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

	"github.com/cloudfoundry/libbuildpack/cutlass/execution"
)

func main() {
	var logger = libbuildpack.NewLogger(os.Stdout)
	if len(os.Args) != 6 {
		logger.Error("incorrect number of arguments")
		os.Exit(1)
	}

	if err := finalize(logger); err != nil {
		logger.Error("Failed finalize step: %s", err)
		os.Exit(1)
	}
}

func finalize(logger *libbuildpack.Logger) error {
	v2AppDir := os.Args[1]
	v2CacheDir := os.Args[2]
	v2DepsDir := os.Args[3]
	v2DepsIndex := os.Args[4]
	profileDir := os.Args[5]

	defer os.RemoveAll(shims.V3StoredOrderDir)
	defer os.RemoveAll(shims.V3BuildpacksDir)
	defer os.RemoveAll(shims.V3MetadataDir)

	tempDir, err := ioutil.TempDir("", "temp")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := os.MkdirAll(shims.V3MetadataDir, 0777); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(shims.V3AppDir, ".cloudfoundry"), 0777); err != nil {
		return err
	}

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		return err
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		return err
	}

	installer := shims.NewCNBInstaller(manifest)

	detectExecPath := filepath.Join(tempDir, shims.V3Detector)
	detectExecutable := execution.NewExecutable(detectExecPath, lager.NewLogger("detect"))

	finalizeExecPath := filepath.Join(tempDir, shims.V3Builder)
	finalizeExecutable := execution.NewExecutable(finalizeExecPath, lager.NewLogger("finalize"))

	finalizer := shims.Finalizer{
		V2AppDir:        v2AppDir,
		V3AppDir:        shims.V3AppDir,
		V2DepsDir:       v2DepsDir,
		V2CacheDir:      v2CacheDir,
		V3LayersDir:     shims.V3LayersDir,
		V3BuildpacksDir: shims.V3BuildpacksDir,
		DepsIndex:       v2DepsIndex,
		OrderDir:        shims.V3StoredOrderDir,
		OrderMetadata:   filepath.Join(shims.V3MetadataDir, "order.toml"),
		GroupMetadata:   filepath.Join(shims.V3MetadataDir, "group.toml"),
		PlanMetadata:    filepath.Join(shims.V3MetadataDir, "plan.toml"),
		V3LifecycleDir:  tempDir,
		V3LauncherDir:   filepath.Join(shims.V3AppDir, ".cloudfoundry"), // We need to put the launcher binary somewhere in the droplet so it can run at launch. Can we put this here? If it is in depsDir/launcher could overlap with a v3 buildpack called "launcher"
		ProfileDir:      profileDir,
		Detector: shims.Detector{
			AppDir:          shims.V3AppDir,
			V3LifecycleDir:  tempDir,
			V3BuildpacksDir: shims.V3BuildpacksDir,
			OrderMetadata:   filepath.Join(shims.V3MetadataDir, "order.toml"),
			GroupMetadata:   filepath.Join(shims.V3MetadataDir, "group.toml"),
			PlanMetadata:    filepath.Join(shims.V3MetadataDir, "plan.toml"),
			Installer:       installer,
			Environment:     cloudnative.NewEnvironment(),
			Executor:        detectExecutable,
		},
		Installer:   installer,
		Manifest:    manifest,
		Logger:      logger,
		Executable:  finalizeExecutable,
		Environment: cloudnative.NewEnvironment(),
	}

	return finalizer.Finalize()
}
