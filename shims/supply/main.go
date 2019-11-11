package main

import (
	"os"
	"time"

	"github.com/cloudfoundry/cnb2cf/shims"
	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	var logger = libbuildpack.NewLogger(os.Stdout)
	if len(os.Args) != 5 {
		logger.Error("Incorrect number of arguments")
		os.Exit(1)
	}

	if err := supply(logger); err != nil {
		logger.Error("Failed supply step: %s", err.Error())
		os.Exit(1)
	}
}

func supply(logger *libbuildpack.Logger) error {
	v2AppDir := os.Args[1]
	v2CacheDir := os.Args[2]
	v2DepsDir := os.Args[3]
	depsIndex := os.Args[4]

	buildpackDir, err := libbuildpack.GetBuildpackDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(shims.V3AppDir, 0777); err != nil {
		return err
	}

	if err := os.MkdirAll(shims.V3StoredOrderDir, 0777); err != nil {
		return err
	}

	err = os.MkdirAll(shims.V3BuildpacksDir, 0777)
	if err != nil {
		return err
	}

	manifest, err := libbuildpack.NewManifest(buildpackDir, logger, time.Now())
	if err != nil {
		return err
	}

	supplier := shims.Supplier{
		V2AppDir:        v2AppDir,
		V3AppDir:        shims.V3AppDir,
		V2DepsDir:       v2DepsDir,
		V2CacheDir:      v2CacheDir,
		DepsIndex:       depsIndex,
		V2BuildpackDir:  buildpackDir,
		V3BuildpacksDir: shims.V3BuildpacksDir,
		OrderDir:        shims.V3StoredOrderDir,
		Installer:       shims.NewCNBInstaller(manifest, libbuildpack.NewInstaller(manifest)),
		Manifest:        manifest,
		Logger:          logger,
	}

	return supplier.Supply()
}
